package controller

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/xenvoid404/neko-tunneling/api/dto"
	"github.com/xenvoid404/neko-tunneling/api/service"
	"github.com/xenvoid404/neko-tunneling/config"
	"github.com/xenvoid404/neko-tunneling/pkg/provision"
	"github.com/xenvoid404/neko-tunneling/pkg/utils"
	"github.com/xenvoid404/neko-tunneling/pkg/validator"
)

func handleTrial(cfg *config.Config, protocol string) fiber.Handler {
	return func(c fiber.Ctx) error {
		validProtocols := map[string]bool{"ssh": true, "vmess": true, "vless": true, "trojan": true}
		if !validProtocols[protocol] {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(&dto.ErrorRes{
				Success: false,
				Message: "no",
				Errors: fiber.Map{
					"protocol": []string{"protocol tidak valid"},
				},
			})
		}

		var req dto.TrialReq
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(&dto.ErrorRes{
				Success: false,
				Message: "body payload tidak valid",
			})
		}

		if errs := validator.ValidateStruct(&req); errs != nil {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(&dto.ErrorRes{
				Success: false,
				Message: "no",
				Errors:  errs,
			})
		}

		account, err := service.CreateTrial(c.Context(), cfg, protocol, req.Expired)
		if err != nil {
			slog.Error("Gagal membuat akun trial",
				slog.String("protocol", protocol),
				slog.Any("error", err))
			return c.Status(fiber.StatusInternalServerError).JSON(&dto.ErrorRes{
				Success: false,
				Message: "pembuatan akun gagal",
			})
		}

		hostname := utils.ReadFile(cfg.CacheDomainPath)
		isp := utils.ReadFile(cfg.CacheISPPath)
		city := utils.ReadFile(cfg.CacheCityPath)
		expiredStr := account.ExpiredAt.Format("2006-01-02 15:04:05")

		return c.Status(fiber.StatusOK).JSON(&dto.SuccessRes{
			Success: true,
			Message: "ok",
			Data:    buildTrialRes(protocol, hostname, isp, city, account.Username, account.Password, expiredStr),
		})
	}
}

func buildTrialRes(protocol, hostname, isp, city, username, password, expired string) interface{} {
	switch protocol {
	case "ssh":
		return dto.SSHData{
			Hostname: hostname,
			ISP:      isp,
			City:     city,
			Username: username,
			Password: password,
			Expired:  expired,
			Port: dto.PortData{
				None:  []string{"80", "8080"},
				TLS:   []string{"443", "444", "8443"},
				UDPGW: []string{"7100", "7200", "7300", "7400", "7500", "7600"},
			},
			PayloadWS: dto.PayloadWSData{
				PayloadCDN:      "GET / HTTP/1.1[crlf]Host: [host_port][crlf]User-Agent: [ua][crlf]Upgrade: websocket[crlf][crlf]",
				PayloadWithPath: "GET /worryfree/ssh HTTP/1.1[crlf]Host: BUG[crlf]User-Agent: [ua][crlf]Upgrade: websocket[crlf][crlf]",
			},
		}
	case "vmess":
		return dto.VmessData{
			Hostname: hostname,
			ISP:      isp,
			City:     city,
			Username: username,
			Password: password,
			Expired:  expired,
			Port:     dto.PortData{None: []string{"80"}, TLS: []string{"443"}},
			Path:     dto.PathData{WS: "vmess-ws", Upgrade: "vmess-up", GRPC: "vmess-grpc"},
			Link:     buildTrialXrayLink("vmess", username, hostname, password, "vmess-ws", "vmess-up", "vmess-grpc"),
		}
	case "vless":
		return dto.VlessData{
			Hostname: hostname,
			ISP:      isp,
			City:     city,
			Username: username,
			Password: password,
			Expired:  expired,
			Port:     dto.PortData{None: []string{"80"}, TLS: []string{"443"}},
			Path:     dto.PathData{WS: "vless-ws", Upgrade: "vless-up", GRPC: "vless-grpc"},
			Link:     buildTrialXrayLink("vless", username, hostname, password, "vless-ws", "vless-up", "vless-grpc"),
		}
	case "trojan":
		return dto.TrojanData{
			Hostname: hostname,
			ISP:      isp,
			City:     city,
			Username: username,
			Password: password,
			Expired:  expired,
			Port:     dto.PortData{TLS: []string{"443"}},
			Path:     dto.PathData{WS: "trojan-ws", Upgrade: "trojan-up", GRPC: "trojan-grpc"},
			Link:     buildTrialXrayLink("trojan", username, hostname, password, "trojan-ws", "trojan-up", "trojan-grpc"),
		}
	default:
		return nil
	}
}

func buildTrialXrayLink(protocol, username, hostname, secret, wsPath, upPath, grpcPath string) dto.XrayLinkData {
	switch protocol {
	case "vmess":
		none, _ := provision.GenerateVmess(username, hostname, 80, secret, "ws", wsPath, false)
		tls, _ := provision.GenerateVmess(username, hostname, 443, secret, "ws", wsPath, true)
		grpc, _ := provision.GenerateVmess(username, hostname, 443, secret, "grpc", grpcPath, true)
		upNone, _ := provision.GenerateVmess(username, hostname, 80, secret, "httpupgrade", upPath, false)
		upTLS, _ := provision.GenerateVmess(username, hostname, 443, secret, "httpupgrade", upPath, true)
		return dto.XrayLinkData{None: none, TLS: tls, GRPC: grpc, UpNone: upNone, UpTLS: upTLS}
	case "vless":
		return dto.XrayLinkData{
			None:   provision.GenerateVless(username, hostname, 80, secret, "ws", wsPath, false),
			TLS:    provision.GenerateVless(username, hostname, 443, secret, "ws", wsPath, true),
			GRPC:   provision.GenerateVless(username, hostname, 443, secret, "grpc", grpcPath, true),
			UpNone: provision.GenerateVless(username, hostname, 80, secret, "httpupgrade", upPath, false),
			UpTLS:  provision.GenerateVless(username, hostname, 443, secret, "httpupgrade", upPath, true),
		}
	case "trojan":
		return dto.XrayLinkData{
			TLS:    provision.GenerateTrojan(username, hostname, 443, secret, "ws", wsPath, true),
			GRPC:   provision.GenerateTrojan(username, hostname, 443, secret, "grpc", grpcPath, true),
			UpNone: provision.GenerateTrojan(username, hostname, 80, secret, "httpupgrade", upPath, false),
			UpTLS:  provision.GenerateTrojan(username, hostname, 443, secret, "httpupgrade", upPath, true),
		}
	default:
		return dto.XrayLinkData{}
	}
}

// TrialSSH godoc
// @Summary   Trial Akun SSH
// @Tags      Trial Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     expired formData int true "Durasi Trial (Menit, maks 1440)"
// @Success   200 {object} dto.SuccessRes{data=dto.SSHData} "Berhasil membuat akun trial SSH"
// @Failure   400 {object} dto.ErrorRes "Bad Request - Payload body tidak valid"
// @Failure   422 {object} dto.ErrorRes "Unprocessable Entity - Validasi form gagal"
// @Failure   500 {object} dto.ErrorRes "Internal Server Error - Gagal eksekusi ke server"
// @Security  BearerAuth
// @Router    /vps/trial/ssh [post]
func TrialSSH(cfg *config.Config) fiber.Handler {
	return handleTrial(cfg, "ssh")
}

// TrialVmess godoc
// @Summary   Trial Akun Vmess
// @Tags      Trial Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     expired formData int true "Durasi Trial (Menit, maks 1440)"
// @Success   200 {object} dto.SuccessRes{data=dto.VmessData} "Berhasil membuat akun trial Vmess"
// @Failure   400 {object} dto.ErrorRes "Bad Request - Payload body tidak valid"
// @Failure   422 {object} dto.ErrorRes "Unprocessable Entity - Validasi form gagal"
// @Failure   500 {object} dto.ErrorRes "Internal Server Error - Gagal eksekusi ke server"
// @Security  BearerAuth
// @Router    /vps/trial/vmess [post]
func TrialVmess(cfg *config.Config) fiber.Handler {
	return handleTrial(cfg, "vmess")
}

// TrialVless godoc
// @Summary   Trial Akun Vless
// @Tags      Trial Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     expired formData int true "Durasi Trial (Menit, maks 1440)"
// @Success   200 {object} dto.SuccessRes{data=dto.VlessData} "Berhasil membuat akun trial Vless"
// @Failure   400 {object} dto.ErrorRes "Bad Request - Payload body tidak valid"
// @Failure   422 {object} dto.ErrorRes "Unprocessable Entity - Validasi form gagal"
// @Failure   500 {object} dto.ErrorRes "Internal Server Error - Gagal eksekusi ke server"
// @Security  BearerAuth
// @Router    /vps/trial/vless [post]
func TrialVless(cfg *config.Config) fiber.Handler {
	return handleTrial(cfg, "vless")
}

// TrialTrojan godoc
// @Summary   Trial Akun Trojan
// @Tags      Trial Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     expired formData int true "Durasi Trial (Menit, maks 1440)"
// @Success   200 {object} dto.SuccessRes{data=dto.TrojanData} "Berhasil membuat akun trial Trojan"
// @Failure   400 {object} dto.ErrorRes "Bad Request - Payload body tidak valid"
// @Failure   422 {object} dto.ErrorRes "Unprocessable Entity - Validasi form gagal"
// @Failure   500 {object} dto.ErrorRes "Internal Server Error - Gagal eksekusi ke server"
// @Security  BearerAuth
// @Router    /vps/trial/trojan [post]
func TrialTrojan(cfg *config.Config) fiber.Handler {
	return handleTrial(cfg, "trojan")
}
