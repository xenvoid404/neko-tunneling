package mux

import (
	"bufio"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type muxConn struct {
	net.Conn
	targetSSH string
}

func (m *muxConn) CloseWrite() error {
	if cw, ok := m.Conn.(closeWriter); ok {
		return cw.CloseWrite()
	}
	return nil
}

type peekConn struct {
	net.Conn
	reader *bufio.Reader
}

func newPeekConn(conn net.Conn) *peekConn {
	return &peekConn{Conn: conn, reader: bufio.NewReaderSize(conn, 4096)}
}

func (c *peekConn) peekPrefix() ([]byte, error) {
	_ = c.Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	defer func() { _ = c.Conn.SetReadDeadline(time.Time{}) }()
	return c.reader.Peek(4)
}

func (c *peekConn) Read(p []byte) (int, error) { return c.reader.Read(p) }

func (c *peekConn) CloseWrite() error {
	if cw, ok := c.Conn.(closeWriter); ok {
		return cw.CloseWrite()
	}
	return nil
}

type singleConnListener struct {
	conn net.Conn
	done atomic.Bool
}

func (l *singleConnListener) Accept() (net.Conn, error) {
	if l.done.Swap(true) {
		return nil, http.ErrServerClosed
	}
	return l.conn, nil
}

func (l *singleConnListener) Close() error { return nil }

func (l *singleConnListener) Addr() net.Addr { return l.conn.LocalAddr() }

const bufSize = 32 * 1024

var bufPool = sync.Pool{New: func() any { b := make([]byte, bufSize); return &b }}

func getBuf() *[]byte  { return bufPool.Get().(*[]byte) }
func putBuf(b *[]byte) { bufPool.Put(b) }

type closeWriter interface{ CloseWrite() error }

func closeWrite(c net.Conn) {
	if cw, ok := c.(closeWriter); ok {
		_ = cw.CloseWrite()
	}
}

func runPipes(c1, c2 net.Conn, mode string) {
	defer c1.Close()
	defer c2.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	var sent, received int64

	go func() {
		defer wg.Done()
		buf := getBuf()
		n, _ := io.CopyBuffer(c2, c1, *buf)
		sent = n
		putBuf(buf)
		closeWrite(c2)
	}()

	go func() {
		defer wg.Done()
		buf := getBuf()
		n, _ := io.CopyBuffer(c1, c2, *buf)
		received = n
		putBuf(buf)
		closeWrite(c1)
	}()

	wg.Wait()

	slog.Debug("Sesi tunnel selesai",
		slog.String("metode", mode),
		slog.Int64("bytes_terkirim", sent),
		slog.Int64("bytes_diterima", received))
}
