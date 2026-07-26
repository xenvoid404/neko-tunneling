package controller

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/xenvoid404/neko-tunneling/dto"
	"github.com/xenvoid404/neko-tunneling/pkg/logger"
	"github.com/xenvoid404/neko-tunneling/pkg/utils"
	"github.com/xenvoid404/neko-tunneling/pkg/validator"
	"github.com/xenvoid404/neko-tunneling/repository"
)

var log = logger.CreateLogger()

func handleTrial(protocol string) fiber.Handler {
	return func(c fiber.Ctx) error {
		validProtocols := map[string]bool{"ssh": true, "vmess": true, "vless": true, "trojan": true}
		if !validProtocols[protocol] {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(&dto.ErrorRes{
				Syccess: false,
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

		username := utils.RandomUsername()
		password := utils.RandomPassword(protocol)
		expiredAt := time.Now().Add(time.Duration(req.Expired) * time.Minute)

		if protocol == "ssh" {
			if err := utils.AddSSHUser(username, password, expiredAt); err != nil {
				log.Error("Gagal membuat user Linux",
					slog.String("username", username),
					slog.Any("error", err))
				return c.Status(fiber.StatusInternalServerError).JSON(&dto.ErrorRes{
					Success: false,
					Message: "pembuatan akun gagal",
				})
			}
		} else {
			if err := utils.AddXrayUser(cfg.XrayAPIAddr, protocol, password, username); err != nil {
				log.Error("Gagal membuat user Xray",
					slog.String("protocol", protocol),
					slog.String("username", username),
					slog.Any("error", err))
				return c.Status(fiber.StatusInternalServerError).JSON(&dto.ErrorRes{
					Success: false,
					Message: "pembuatan akun gagal",
				})
			}
		}

		user := repository.User{
			Protocol:   protocol,
			Username:   username,
			Password:   password,
			LimitIP:    0,
			LimitQuota: 0,
			Status:     "active",
			ExpiredAt:  expiredAt.Format("2006-01-02 15:04:05"),
		}

		if err := repository.CreateUser(c.Context(), user); err != nil {
			log.Error("Gagal menyimpan ke database",
				slog.Any("error", err))
			if protocol == "ssh" {
				if delErr := utils.DeleteSSHUser(username); delErr != nil {
					log.Error("Gagal rollback user SSH",
						slog.Any("error", delErr))
				}
			}

			return c.Status(fiber.StatusInternalServerError).JSON(&dto.ErrorRes{
				Success: false,
				Message: "Pembuatan akun gagal",
			})
		}

		hostname := sysutil.ReadFile("/var/lib/nekotun/cache/domain")
		isp := sysutil.ReadFile("/var/lib/nekotun/cache/isp")
		city := sysutil.ReadFile("/var/lib/nekotun/cache/city")

		switch protocol {
		case "ssh":
			return c.Status(fiber.StatusOK).JSON(&dto.SuccessRes{
				Success: true,
				Message: "ok",
				Data: dto.SSHData{
					Hostname: hostname,
					ISP:      isp,
					City:     city,
					Username: username,
					Password: password,
					Expired:  expiredAt.Format("2006-01-02 15:04:05"),
					Port: dto.PortData{
						None:  []string{"80", "8080"},
						TLS:   []string{"443", "444", "8443"},
						UDPGW: []string{"7100", "7200", "7300", "7400", "7500", "7600"},
					},
					PayloadWS: dto.PayloadWSData{
						PayloadCDN:      "GET / HTTP/1.1[crlf]Host: [host_port][crlf]User-Agent: [ua][crlf]Upgrade: websocket[crlf][crlf]",
						PayloadWithPath: "GET /worryfree/ssh HTTP/1.1[crlf]Host: BUG[crlf]User-Agent: [ua][crlf]Upgrade: websocket[crlf][crlf]",
					},
				},
			})
		case "vmess":
			return c.Status(fiber.StatusOK).JSON(&dto.SuccessRes{
				Success: true,
				Message: "ok",
				Data: dto.VmessData{
					Hostname: hostname,
					ISP:      isp,
					City:     city,
					Username: username,
					Password: password,
					Expired:  expiredAt.Format("2006-01-02 15:04:05"),
					Port: dto.PortData{
						None: []string{"80"},
						TLS:  []string{"443"},
					},
					Path: dto.PathData{
						WS:      "vmess-ws",
						Upgrade: "vmess-up",
						GRPC:    "vmess-grpc",
					},
					Link: dto.XrayLinkData{
						None:   utils.GenerateVMess(username, hostname, 80, password, "ws", "vmess-ws", false),
						TLS:    utils.GenerateVMess(username, hostname, 443, password, "ws", "vmess-ws", true),
						GRPC:   utils.GenerateVMess(username, hostname, 443, password, "grpc", "vmess-grpc", true),
						UpNone: utils.GenerateVMess(username, hostname, 80, password, "httpupgrade", "vmess-up", false),
						UpTLS:  utils.GenerateVMess(username, hostname, 443, password, "httpupgrade", "vmess-up", true),
					},
				},
			})
		case "vless":
			return c.Status(fiber.StatusOK).JSON(&dto.SuccessRes{
				Success: true,
				Message: "ok",
				Data: dto.VlessData{
					Hostname: hostname,
					ISP:      isp,
					City:     city,
					Username: username,
					Password: password,
					Expired:  expiredAt.Format("2006-01-02 15:04:05"),
					Port: dto.PortData{
						None: []string{"80"},
						TLS:  []string{"443"},
					},
					Path: dto.PathData{
						WS:      "vless-ws",
						Upgrade: "vless-up",
						GRPC:    "vless-grpc",
					},
					Link: dto.XrayLinkData{
						None:   utils.GenerateVless(username, hostname, 80, password, "ws", "vless-ws", false),
						TLS:    utils.GenerateVless(username, hostname, 443, password, "ws", "vless-ws", true),
						GRPC:   utils.GenerateVless(username, hostname, 443, password, "grpc", "vless-grpc", true),
						UpNone: utils.GenerateVless(username, hostname, 80, password, "httpupgrade", "vless-up", false),
						UpTLS:  utils.GenerateVless(username, hostname, 443, password, "httpupgrade", "vless-up", true),
					},
				},
			})
		case "trojan":
			return c.Status(fiber.StatusOK).JSON(&dto.SuccessRes{
				Success: true,
				Message: "ok",
				Data: dto.TrojanData{
					Hostname: hostname,
					ISP:      isp,
					City:     city,
					Username: username,
					Password: password,
					Expired:  expiredAt.Format("2006-01-02 15:04:05"),
					Port: dto.PortData{
						TLS: []string{"443"},
					},
					Path: dto.PathData{
						WS:      "vless-ws",
						Upgrade: "vless-up",
						GRPC:    "vless-grpc",
					},
					Link: dto.XrayLinkData{
						TLS:    utils.GenerateVless(username, hostname, 443, password, "ws", "vless-ws", true),
						GRPC:   utils.GenerateVless(username, hostname, 443, password, "grpc", "vless-grpc", true),
						UpNone: utils.GenerateVless(username, hostname, 80, password, "httpupgrade", "vless-up", false),
						UpTLS:  utils.GenerateVless(username, hostname, 443, password, "httpupgrade", "vless-up", true),
					},
				},
			})
		}
	}
}

// TrialSSH godoc
// @Summary   Trial Akun Ssh
// @Tags      Trial Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     expired formData int true "Durasi Trial (Menit)"
// @Success   200 {object} dto.SuccessRes{data=dto.SSHData} "Berhasil membuat akun trial SSH"
// @Failure   400 {object} dto.ErrorRes "Bad Request - Payload body tidak valid"
// @Failure   422 {object} dto.ErrorRes "Unprocessable Entity - Validasi form gagal"
// @Failure   500 {object} dto.ErrorRes "Internal Server Error - Gagal eksekusi ke server"
// @Router    /vps/trial/ssh [post]
func TrialSSH() fiber.Handler {
	return handleTrial("ssh")
}

// TrialVmess godoc
// @Summary   Trial Akun Vmess
// @Tags      Trial Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     expired formData int true "Durasi Trial (Menit)"
// @Router    /vps/trial/vmess [post]
func TrialVmess() fiber.Handler {
	return handleTrial("vmess")
}

// TrialVless godoc
// @Summary   Trial Akun Vless
// @Tags      Trial Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     expired formData int true "Durasi Trial (Menit)"
// @Router    /vps/trial/vless [post]
func TrialVless() fiber.Handler {
	return handleTrial("vless")
}

// TrialTrojan godoc
// @Summary   Trial Akun Trojan
// @Tags      Trial Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     expired formData int true "Durasi Trial (Menit)"
// @Router    /vps/trial/trojan [post]
func TrialTrojan() fiber.Handler {
	return handleTrial("trojan")
}
