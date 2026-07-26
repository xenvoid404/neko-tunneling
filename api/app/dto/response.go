package dto

type SuccessRes struct {
	Success bool        `json:"success" example:"true"`
	Message string      `json:"message" example:"ok"`
	Data    interface{} `json:"data,omitempty"`
}

type ErrorRes struct {
	Success bool        `json:"success" example:"false"`
	Message string      `json:"message" example:"no"`
	Errors  interface{} `json:"errors,omitempty"`
}

type SSHData struct {
	Hostname  string        `json:"hostname" example:"domain.com"`
	ISP       string        `json:"isp" example:"Neko Tunneling ltd"`
	City      string        `json:"city" example:"Jakarta"`
	Username  string        `json:"username" example:"neko123"`
	Password  string        `json:"password" example:"neko123"`
	Expired   string        `json:"expired" example:"2006-01-02 15:04:05"`
	Port      PortData      `json:"port"`
	PayloadWS PayloadWSData `json:"payload_ws"`
}

type VmessData struct {
	Hostname string       `json:"hostname" example:"domain.com"`
	ISP      string       `json:"isp" example:"Neko Tunneling ltd"`
	City     string       `json:"city" example:"Jakarta"`
	Username string       `json:"username" example:"neko123"`
	Password string       `json:"password" example:"neko123"`
	Expired  string       `json:"expired" example:"2006-01-02 15:04:05"`
	Port     PortData     `json:"port"`
	Path     PathData     `json:"path"`
	Link     XrayLinkData `json:"link"`
}

type VlessData struct {
	Hostname string       `json:"hostname" example:"domain.com"`
	ISP      string       `json:"isp" example:"Neko Tunneling ltd"`
	City     string       `json:"city" example:"Jakarta"`
	Username string       `json:"username" example:"neko123"`
	Password string       `json:"password" example:"neko123"`
	Expired  string       `json:"expired" example:"2006-01-02 15:04:05"`
	Port     PortData     `json:"port"`
	Path     PathData     `json:"path"`
	Link     XrayLinkData `json:"link"`
}

type TrojanData struct {
	Hostname string       `json:"hostname" example:"domain.com"`
	ISP      string       `json:"isp" example:"Neko Tunneling ltd"`
	City     string       `json:"city" example:"Jakarta"`
	Username string       `json:"username" example:"neko123"`
	Password string       `json:"password" example:"neko123"`
	Expired  string       `json:"expired" example:"2006-01-02 15:04:05"`
	Port     PortData     `json:"port"`
	Path     PathData     `json:"path"`
	Link     XrayLinkData `json:"link"`
}

type PortData struct {
	None  []string `json:"none,omitempty"`
	TLS   []string `json:"tls,omitempty"`
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
