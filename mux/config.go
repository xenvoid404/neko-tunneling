package main

import (
	"net/http/httputil"
	"time"
)

type listener struct {
	addr       string
	isTLS      bool
	sshBackend string
}
type route struct {
	path   string
	target string
	proxy  *httputil.ReverseProxy
	isGRPC bool
}

type config struct {
	certFile            string
	keyFile             string
	dropbearAddr        string
	openSSHAddr         string
	dialTimeout         time.Duration
	keepAliveTimeout    time.Duration
	maxIdleConns        int
	maxIdleConnsPerHost int
	idleTimeout         time.Duration
	readTimeout         time.Duration
	listeners           []listener
	routes              []route
}

func getConfig() *config {
	return &config{
		certFile:            "/etc/nekotun/certs/fullchain.cer",
		keyFile:             "/etc/nekotun/certs/private.key",
		dropbearAddr:        "127.0.0.1:90",
		openSSHAddr:         "127.0.0.1:22",
		dialTimeout:         3 * time.Second,
		keepAliveTimeout:    30 * time.Second,
		maxIdleConns:        100,
		maxIdleConnsPerHost: 20,
		idleTimeout:         90 * time.Second,
		readTimeout:         5 * time.Second,
		listeners: []listener{
			{addr: ":80", isTLS: false, sshBackend: "dropbear"},
			{addr: ":8080", isTLS: false, sshBackend: "dropbear"},
			{addr: ":443", isTLS: true, sshBackend: "dropbear"},
			{addr: ":444", isTLS: true, sshBackend: "openssh"},
			{addr: ":8443", isTLS: true, sshBackend: "dropbear"},
		},
		routes: []route{
			{path: "/vmess-ws", target: "127.0.0.1:1054", isGRPC: false},
			{path: "/vless-ws", target: "127.0.0.1:1057", isGRPC: false},
			{path: "/trojan-ws", target: "127.0.0.1:1060", isGRPC: false},
			{path: "/vmess-up", target: "127.0.0.1:1056", isGRPC: false},
			{path: "/vless-up", target: "127.0.0.1:1059", isGRPC: false},
			{path: "/trojan-up", target: "127.0.0.1:1062", isGRPC: false},
			{path: "/vmess-grpc", target: "127.0.0.1:1055", isGRPC: true},
			{path: "/vless-grpc", target: "127.0.0.1:1058", isGRPC: true},
			{path: "/trojan-grpc", target: "127.0.0.1:1061", isGRPC: true},
			{path: "/vps", target: "127.0.0.1:3000", isGRPC: false},
		},
	}
}
