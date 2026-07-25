package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http2"
)

type server struct {
	tlsConfig  *tls.Config
	httpServer *http.Server
	routes     []route
}

func newServer() (*server, error) {
	cert, err := tls.LoadX509KeyPair(cfg.certFile, cfg.keyFile)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat sertifikat TLS: %w", err)
	}

	server := &server{
		tlsConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{"http/1.1", "h2"},
		},
	}

	httpRT := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   cfg.dialTimeout,
			KeepAlive: cfg.keepAliveTimeout,
		}).DialContext,
		MaxIdleConns:        cfg.maxIdleConns,
		MaxIdleConnsPerHost: cfg.maxIdleConnsPerHost,
		IdleConnTimeout:     cfg.idleTimeout,
	}

	grpcRT := &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
			return (&net.Dialer{
				Timeout:   cfg.dialTimeout,
				KeepAlive: cfg.keepAliveTimeout,
			}).DialContext(ctx, network, addr)
		},
	}

	for _, r := range cfg.routes {
		transport := http.RoundTripper(httpRT)
		if r.isGRPC {
			transport = grpcRT
		}

		proxy := httputil.NewSingleHostReverseProxy(&url.URL{Scheme: "http", Host: r.target})
		proxy.FlushInterval = -1
		proxy.Transport = transport
		proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
			log.Error("Gagal meneruskan request ke backend Xray",
				slog.String("path", r.path),
				slog.String("target", r.target),
				slog.String("remote_addr", req.RemoteAddr),
				slog.Any("error", err))
			w.WriteHeader(http.StatusBadRequest)
		}

		server.routes = append(server.routes, route{
			path:   r.path,
			target: r.target,
			proxy:  proxy,
			isGRPC: r.isGRPC,
		})
	}

	server.httpServer = &http.Server{
		Handler:     server,
		ReadTimeout: cfg.readTimeout,
	}

	log.Info("Multiplexer berhasil diinisialisasi",
		slog.Int("jumlah_route", len(server.routes)),
		slog.Int("jumlah_listener", len(cfg.listeners)))

	return server, nil
}

func (s *server) start(ctx context.Context) error {
	var wg sync.WaitGroup
	var listeners []net.Listener

	for _, spec := range cfg.listeners {
		ln, err := net.Listen("tcp", spec.addr)
		if err != nil {
			return fmt.Errorf("%s gagal mendengarkan: %w", spec.addr, err)
		}

		listeners = append(listeners, ln)
		log.Info("Mendengarkan koneksi",
			slog.String("addr", spec.addr),
			slog.Bool("tls", spec.isTLS),
			slog.String("ssh_backend", spec.sshBackend))

		wg.Add(1)
		go func(l net.Listener, sp listener) {
			defer wg.Done()
			s.acceptLoop(l, sp)
		}(ln, spec)
	}

	<-ctx.Done()
	log.Info("Menerima sinyal shutdown, menutup semua listener...")
	for _, ln := range listeners {
		_ = ln.Close()
	}

	wg.Wait()
	log.Info("Multiplexer berhenti dengan elegan")
	return nil
}

func (s *server) acceptLoop(ln net.Listener, spec listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Warn("Gagal menerima koneksi baru",
					slog.String("addr", spec.addr),
					slog.Any("error", err))
				continue
			}
			return
		}

		go s.processConnection(conn, spec)
	}
}

func (s *server) processConnection(conn net.Conn, spec listener) {
	targetSSH := cfg.dropbearAddr
	if spec.sshBackend == "openssh" {
		targetSSH = cfg.openSSHAddr
	}

	if !spec.isTLS {
		s.sniffProtocol(conn, targetSSH, "SSH Direct (TCP Mentah)")
		return
	}

	tlsConn, isHTTP2, err := s.performTLSHandshake(conn)
	if err != nil {
		_ = conn.Close()
		return
	}

	if isHTTP2 {
		log.Debug("ALPN h2 terdeteksi, diarahkan langsung sebagai HTTP/2 (kemungkinan gRPC)",
			slog.String("remote_addr", conn.RemoteAddr().String()))
		s.routeToHTTPServer(tlsConn, targetSSH)
		return
	}

	s.sniffProtocol(tlsConn, targetSSH, "SSH over TLS (Stunnel)")
}

func (s *server) performTLSHandshake(conn net.Conn) (*tls.Conn, bool, error) {
	tlsConn := tls.Server(conn, s.tlsConfig)
	_ = tlsConn.SetDeadline(time.Now().Add(cfg.readTimeout))
	err := tlsConn.Handshake()
	_ = tlsConn.SetDeadline(time.Time{})
	if err != nil {
		log.Debug("Handshake TLS gagal",
			slog.String("remote_addr", conn.RemoteAddr().String()),
			slog.Any("error", err))
		return nil, false, err
	}

	state := tlsConn.ConnectionState()
	isHTTP2 := (state.NegotiatedProtocol == "h2")

	return tlsConn, isHTTP2, nil
}

func (s *server) sniffProtocol(conn net.Conn, targetSSH, mode string) {
	pc := newPeekConn(conn)
	peekedData, err := pc.peekPrefix()
	if err != nil {
		log.Debug("Menutup koneksi: gagal membaca byte pertama (idle/timeout)",
			slog.String("remote_addr", conn.RemoteAddr().String()),
			slog.Any("error", err))
		_ = conn.Close()
		return
	}

	if bytes.HasPrefix(peekedData, []byte("SSH-")) {
		s.bridgeConnection(pc, targetSSH, mode)
		return
	}

	s.routeToHTTPServer(pc, targetSSH)
}

func (s *server) routeToHTTPServer(clientConn net.Conn, targetSSH string) {
	mc := &muxConn{Conn: clientConn, targetSSH: targetSSH}
	dummyListener := &singleConnListener{conn: mc}
	err := s.httpServer.Serve(dummyListener)
	if err != nil && err != http.ErrServerClosed {
		log.Error("HTTP server berhenti dengan error", slog.Any("error", err))
	}
}

func (s *server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodConnect {
		s.hijackAndBridge(w, "HTTP/1.1 200 Connection Established\r\n\r\n", "HTTP CONNECT")
		return
	}

	for _, r := range s.routes {
		if strings.HasPrefix(req.URL.Path, r.path) {
			log.Debug("Menyalurkan lalu lintas Xray",
				slog.String("path", req.URL.Path),
				slog.String("target", r.target),
				slog.String("remote_addr", req.RemoteAddr))
			r.proxy.ServeHTTP(w, req)
			return
		}
	}

	if strings.EqualFold(req.Header.Get("Upgrade"), "websocket") {
		s.hijackAndBridge(w, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n", "WebSocket")
		return
	}

	log.Debug("Request tidak cocok route manapun, default ke Dropbear (HTTP Payload)",
		slog.String("path", req.URL.Path),
		slog.String("remote_addr", req.RemoteAddr))
	s.hijackAndBridge(w, "HTTP/1.1 200 OK\r\n\r\n", "HTTP Payload (Default)")
}

func (s *server) bridgeConnection(clientConn net.Conn, backendAddr, mode string) {
	backendConn, err := net.DialTimeout("tcp", backendAddr, cfg.dialTimeout)
	if err != nil {
		log.Error("Gagal terhubung ke backend SSH",
			slog.String("metode", mode),
			slog.String("target", backendAddr),
			slog.Any("error", err))
		_ = clientConn.Close()
		return
	}

	log.Info("Menyalurkan koneksi SSH",
		slog.String("metode", mode),
		slog.String("remote_addr", clientConn.RemoteAddr().String()),
		slog.String("target", backendAddr))

	runPipes(clientConn, backendConn, mode)
}

func (s *server) hijackAndBridge(w http.ResponseWriter, initialResponse, mode string) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack tidak didukung", http.StatusInternalServerError)
		return
	}

	clientConn, rw, err := hj.Hijack()
	if err != nil {
		log.Error("Hijack koneksi gagal",
			slog.String("metode", mode),
			slog.Any("error", err))
		return
	}

	targetSSH := cfg.dropbearAddr
	if mc, ok := clientConn.(*muxConn); ok {
		targetSSH = mc.targetSSH
	}

	backendConn, err := net.DialTimeout("tcp", targetSSH, cfg.dialTimeout)
	if err != nil {
		log.Error("Gagal terhubung ke backend SSH dari hijack",
			slog.String("metode", mode),
			slog.String("target", targetSSH),
			slog.Any("error", err))
		_ = clientConn.Close()
		return
	}

	if initialResponse != "" {
		if _, err := rw.WriteString(initialResponse); err != nil {
			log.Error("Gagal mengirim respons awal ke klien",
				slog.String("metode", mode),
				slog.Any("error", err))
			_ = clientConn.Close()
			_ = backendConn.Close()
			return
		}
		if err := rw.Flush(); err != nil {
			log.Error("Gagal flush respons awal ke klien",
				slog.String("metode", mode),
				slog.Any("error", err))
			_ = clientConn.Close()
			_ = backendConn.Close()
			return
		}
	}

	if n := rw.Reader.Buffered(); n > 0 {
		if _, err := io.CopyN(backendConn, rw.Reader, int64(n)); err != nil {
			log.Error("Gagal menguras buffer awal klien",
				slog.String("metode", mode),
				slog.Any("error", err))
			_ = clientConn.Close()
			_ = backendConn.Close()
			return
		}
	}

	log.Info("Menyalurkan koneksi SSH",
		slog.String("metode", mode),
		slog.String("remote_addr", clientConn.RemoteAddr().String()),
		slog.String("target", targetSSH))

	runPipes(clientConn, backendConn, mode)
}
