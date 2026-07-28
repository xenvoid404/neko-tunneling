package model

type ErrorResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors,omitempty"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type SSHData struct {
	Hostname  string        `json:"hostname"`
	ISP       string        `json:"isp"`
	City      string        `json:"city"`
	Username  string        `json:"username"`
	Password  string        `json:"password"`
	Expired   string        `json:"expired"`
	Port      PortData      `json:"port"`
	PayloadWS PayloadWSData `json:"payload_ws"`
}

type VmessData struct {
	Hostname string       `json:"hostname"`
	ISP      string       `json:"isp"`
	City     string       `json:"city"`
	Username string       `json:"username"`
	Password string       `json:"password"`
	Expired  string       `json:"expired"`
	Port     PortData     `json:"port"`
	Path     PathData     `json:"path"`
	Link     XrayLinkData `json:"link"`
}

type VlessData struct {
	Hostname string       `json:"hostname"`
	ISP      string       `json:"isp"`
	City     string       `json:"city"`
	Username string       `json:"username"`
	Password string       `json:"password"`
	Expired  string       `json:"expired"`
	Port     PortData     `json:"port"`
	Path     PathData     `json:"path"`
	Link     XrayLinkData `json:"link"`
}

type TrojanData struct {
	Hostname string       `json:"hostname"`
	ISP      string       `json:"isp"`
	City     string       `json:"city"`
	Username string       `json:"username"`
	Password string       `json:"password"`
	Expired  string       `json:"expired"`
	Port     PortData     `json:"port"`
	Path     PathData     `json:"path"`
	Link     XrayLinkData `json:"link"`
}

type PortData struct {
	None  []string `json:"none,omitempty"`
	TLS   []string `json:"tls,omitempty"`
	GRPC  []string `json:"grpc,omitempty"`
	UDPGW []string `json:"udpgw,omitempty"`
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
	None   string `json:"none,omitempty"`
	TLS    string `json:"tls,omitempty"`
	GRPC   string `json:"grpc,omitempty"`
	UpNone string `json:"upnone,omitempty"`
	UpTLS  string `json:"uptls,omitempty"`
}

type RenewData struct {
	Username string `json:"username"`
	From     string `json:"from"`
	To       string `json:"to"`
}
