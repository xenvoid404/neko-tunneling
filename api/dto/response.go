package dto

import (
	"time"
)

type SuccessRes struct {
	Success bool        `json="success" example="true"`
	Message string      `json="message" example="ok"`
	Data    interface{} `json="data,omitempty"`
}

type ErrorRes struct {
	Success bool        `json="success" example="false"`
	Message string      `json="message" example="no"`
	Errors  interface{} `json="errors,omitempty"`
}

type SSHData struct {
	Hostname  string    `json:"hostname" example:"domain.com"`
	ISP       string    `json:"isp" example:"Neko Tunneling ltd"`
	City      string    `json:"city" example:"Jakarta"`
	Username  string    `json:"username" example:"neko123"`
	Password  string    `json:"password" example:"neko123"`
	Expired   time.Time `json:"expired" example:"2006-01-02 15:04:05"`
	Port      PortData
	PayloadWS PayloadWSData
}

type VmessData struct {
	Hostname string    `json:"hostname" example:"domain.com"`
	ISP      string    `json:"isp" example:"Neko Tunneling ltd"`
	City     string    `json:"city" example:"Jakarta"`
	Username string    `json:"username" example:"neko123"`
	Password string    `json:"password" example:"neko123"`
	Expired  time.Time `json:"expired" example:"2006-01-02 15:04:05"`
	Port     PortData
	Path     PathData
	Link     XrayLinkData
}

type VlessData struct {
	Hostname string    `json:"hostname" example:"domain.com"`
	ISP      string    `json:"isp" example:"Neko Tunneling ltd"`
	City     string    `json:"city" example:"Jakarta"`
	Username string    `json:"username" example:"neko123"`
	Password string    `json:"password" example:"neko123"`
	Expired  time.Time `json:"expired" example:"2006-01-02 15:04:05"`
	Port     PortData
	Path     PathData
	Link     XrayLinkData
}

type TrojanData struct {
	Hostname string    `json:"hostname" example:"domain.com"`
	ISP      string    `json:"isp" example:"Neko Tunneling ltd"`
	City     string    `json:"city" example:"Jakarta"`
	Username string    `json:"username" example:"neko123"`
	Password string    `json:"password" example:"neko123"`
	Expired  time.Time `json:"expired" example:"2006-01-02 15:04:05"`
	Port     PortData
	Path     PathData
	Link     XrayLinkData
}

type PortData struct {
	None  []string `json:"none"`
	TLS   []string `json:"tls"`
	UDPGW []string `json:"udpgw"`
}

type PathData struct {
	WS      string `json:"ws"`
	Upgrade string `json:"upgrade"`
	GRPC    string `json:"grpc"`
}

type PayloadWSData struct {
	PayloadCDN      string `json:"payloadcdn"`
	PayloadWithPath string `json:"payloadwithpath"`
}

type XrayLinkData struct {
	None   string `json:"none"`
	TLS    string `json:"tls"`
	GRPC   string `json:"grpc"`
	UpNone string `json:"upnone"`
	UpTLS  string `jsom:"uptls"`
}
