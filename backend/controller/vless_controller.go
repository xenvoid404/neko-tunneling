package controller

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/xenvoid404/neko-tunneling/config"
	"github.com/xenvoid404/neko-tunneling/model"
	"github.com/xenvoid404/neko-tunneling/pkg/utils"
	"github.com/xenvoid404/neko-tunneling/pkg/validator"
	"github.com/xenvoid404/neko-tunneling/repository"
	"github.com/xenvoid404/neko-tunneling/service"
)

type VlessController struct {
	cfg            *config.Config
	validate       *validator.Validator
	vlessService   *service.VlessService
	userRepository *repository.UserRepository
}

func NewVlessController(cfg *config.Config, validate *validator.Validator, vlessService *service.VlessService, userRepository *repository.UserRepository) *VlessController {
	return &VlessController{
		cfg:            cfg,
		validate:       validate,
		vlessService:   vlessService,
		userRepository: userRepository,
	}
}

// TrialVless godoc
// @Summary   Trial Akun Vless
// @Tags      Trial Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     expired formData int true "Durasi Trial (menit)"
// @Success   200 {object} model.SuccessResponse{data=model.VlessData} "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/trial/vless [post]
func (ctrl *VlessController) Trial(c fiber.Ctx) error {
	var req model.TrialRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&model.ErrorResponse{
			Success: false,
			Message: "body payload tidak valid",
		})
	}

	if errs := ctrl.validate.ValidateStruct(&req); errs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(&model.ErrorResponse{
			Success: false,
			Message: "no",
			Errors:  errs,
		})
	}

	username := utils.RandomUsername()
	password := utils.RandomPassword("vless")
	expired := time.Now().Add(time.Duration(req.Expired) * time.Minute)

	execCtx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	if err := ctrl.vlessService.AddUser(execCtx, username, password); err != nil {
		slog.Error("Gagal membuat trial akun",
			slog.String("protocol", "vless"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pembuatan akun gagal",
		})
	}

	user := model.User{
		Protocol:   "vless",
		Username:   username,
		Password:   password,
		LimitIP:    0,
		LimitQuota: 0,
		Status:     "active",
		ExpiredAt:  expired,
	}

	if err := ctrl.userRepository.Create(c.Context(), user); err != nil {
		slog.Error("Gagal menyimpan akun ke database",
			slog.String("protocol", "vless"),
			slog.String("username", username),
			slog.Any("error", err))

		var rbErr error
		rbErr = ctrl.vlessService.DelUser(execCtx, username)
		if rbErr != nil {
			slog.Error("Rollback gagal, perlu pembersihan manual di sistem",
				slog.String("protocol", "vless"),
				slog.String("username", username),
				slog.Any("error", rbErr))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pembuatan akun gagal",
		})
	}

	hostname := utils.ReadFile(ctrl.cfg.CacheDomainPath)
	isp := utils.ReadFile(ctrl.cfg.CacheISPPath)
	city := utils.ReadFile(ctrl.cfg.CacheCityPath)

	return c.Status(fiber.StatusOK).JSON(&model.SuccessResponse{
		Success: true,
		Message: "ok",
		Data: model.VlessData{
			Hostname: hostname,
			ISP:      isp,
			City:     city,
			Username: username,
			Password: password,
			Expired:  expired.Format("2006-01-02 15:04:05"),
			Port: model.PortData{
				None: []string{"80"},
				TLS:  []string{"443"},
				GRPC: []string{"443"},
			},
			Path: model.PathData{
				WS:      "vless-ws",
				Upgrade: "vless-up",
				GRPC:    "vless-grpc",
			},
			Link: model.XrayLinkData{
				None:   service.GenerateVlessLink(username, hostname, 80, password, "ws", "vless-ws", false),
				TLS:    service.GenerateVlessLink(username, hostname, 443, password, "ws", "vless-ws", true),
				GRPC:   service.GenerateVlessLink(username, hostname, 443, password, "grpc", "vless-grpc", true),
				UpNone: service.GenerateVlessLink(username, hostname, 80, password, "httpupgrade", "vless-up", false),
				UpTLS:  service.GenerateVlessLink(username, hostname, 443, password, "httpupgrade", "vless-up", true),
			},
		},
	})
}

// CreateVless godoc
// @Summary   Create Akun Vless
// @Tags      Create Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     username formData string true "Username akun"
// @Param     password formData string false "Pasword akun (akan dibuat random jika kosong)"
// @Param     limit_ip formData int false "Limit IP (unlimited jika kosong)"
// @Param     limit_quota formData int false "Limit Quota (unlimited jika kosong)"
// @Param     expired formData int true "Durasi masa aktif (hari)"
// @Success   200 {object} model.SuccessResponse{data=model.VlessData} "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/create/vless [post]
func (ctrl *VlessController) Create(c fiber.Ctx) error {
	var req model.CreateRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&model.ErrorResponse{
			Success: false,
			Message: "body payload tidak valid",
		})
	}

	if errs := ctrl.validate.ValidateStruct(&req); errs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(&model.ErrorResponse{
			Success: false,
			Message: "no",
			Errors:  errs,
		})
	}

	username := strings.TrimSpace(strings.ToLower(string(req.Username)))
	execCtx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	existing, err := ctrl.userRepository.FindByUsername(execCtx, username)
	if err != nil {
		slog.Error("Gagal mengecek duplikat username",
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pengecekan username gagal",
		})
	}
	if existing != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(&model.ErrorResponse{
			Success: false,
			Message: "no",
			Errors: fiber.Map{
				"username": []string{"username sudah digunakan"},
			},
		})
	}

	password := utils.RandomPassword("vless")
	if req.Password != nil && *req.Password != "" {
		password = strings.TrimSpace(strings.ToLower(*req.Password))
	}

	limitIP := 0
	if req.LimitIP != nil {
		limitIP = *req.LimitIP
	}

	limitQuota := 0
	if req.LimitQuota != nil {
		limitQuota = *req.LimitQuota
	}

	expired := time.Now().Add(time.Duration(req.Expired) * 24 * time.Hour)

	if err := ctrl.vlessService.AddUser(execCtx, username, password); err != nil {
		slog.Error("Gagal membuat akun",
			slog.String("protocol", "vless"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pembuatan akun gagal",
		})
	}

	user := model.User{
		Protocol:   "vless",
		Username:   username,
		Password:   password,
		LimitIP:    limitIP,
		LimitQuota: limitQuota,
		Status:     "active",
		ExpiredAt:  expired,
	}

	if err := ctrl.userRepository.Create(c.Context(), user); err != nil {
		slog.Error("Gagal menyimpan akun ke database",
			slog.String("protocol", "vless"),
			slog.String("username", username),
			slog.Any("error", err))

		var rbErr error
		rbErr = ctrl.vlessService.DelUser(execCtx, username)
		if rbErr != nil {
			slog.Error("Rollback gagal, perlu pembersihan manual di sistem",
				slog.String("protocol", "vless"),
				slog.String("username", username),
				slog.Any("error", rbErr))
		}
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pembuatan akun gagal",
		})
	}

	hostname := utils.ReadFile(ctrl.cfg.CacheDomainPath)
	isp := utils.ReadFile(ctrl.cfg.CacheISPPath)
	city := utils.ReadFile(ctrl.cfg.CacheCityPath)

	return c.Status(fiber.StatusOK).JSON(&model.SuccessResponse{
		Success: true,
		Message: "ok",
		Data: model.VlessData{
			Hostname: hostname,
			ISP:      isp,
			City:     city,
			Username: username,
			Password: password,
			Expired:  expired.Format("2006-01-02 15:04:05"),
			Port: model.PortData{
				None: []string{"80"},
				TLS:  []string{"443"},
				GRPC: []string{"443"},
			},
			Path: model.PathData{
				WS:      "vless-ws",
				Upgrade: "vless-up",
				GRPC:    "vless-grpc",
			},
			Link: model.XrayLinkData{
				None:   service.GenerateVlessLink(username, hostname, 80, password, "ws", "vless-ws", false),
				TLS:    service.GenerateVlessLink(username, hostname, 443, password, "ws", "vless-ws", true),
				GRPC:   service.GenerateVlessLink(username, hostname, 443, password, "grpc", "vless-grpc", true),
				UpNone: service.GenerateVlessLink(username, hostname, 80, password, "httpupgrade", "vless-up", false),
				UpTLS:  service.GenerateVlessLink(username, hostname, 443, password, "httpupgrade", "vless-up", true),
			},
		},
	})
}

// DeleteVless godoc
// @Summary   Delete Akun Vless
// @Tags      Delete Akun
// @Accept    json
// @Produce   json
// @Param     username path string true "Username akun"
// @Success   200 {object} model.SuccessResponse "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/delete/vless/{username} [delete]
func (ctrl *VlessController) Delete(c fiber.Ctx) error {
	username := strings.TrimSpace(strings.ToLower(c.Params("username")))
	if username == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(&model.ErrorResponse{
			Success: false,
			Message: "no",
			Errors: fiber.Map{
				"username": []string{"username tidak boleh kosong"},
			},
		})
	}

	execCtx, cancel := context.WithTimeout(c.Context(), 10*time.Second)
	defer cancel()

	existing, err := ctrl.userRepository.FindByUsername(execCtx, username)
	if err != nil {
		slog.Error("Gagal mengecek username",
			slog.String("protocol", "vless"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pengecekan username gagal",
		})
	}
	if existing == nil || existing.Protocol != "vless" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(&model.ErrorResponse{
			Success: false,
			Message: "no",
			Errors: fiber.Map{
				"username": []string{"username tidak ditemukan"},
			},
		})
	}

	if err := ctrl.vlessService.DelUser(execCtx, username); err != nil {
		slog.Error("Gagal menghapus user dari sistem",
			slog.String("protocol", "vless"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "gagal menghapus user dari server",
		})
	}

	if err := ctrl.userRepository.DeleteByUsername(execCtx, username); err != nil {
		slog.Error("Gagal menghapus user dari database",
			slog.String("protocol", "vless"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "gagal menghapus data user",
		})
	}

	return c.Status(fiber.StatusOK).JSON(&model.SuccessResponse{
		Success: true,
		Message: "berhasil menghapus user",
	})
}

// RenewVless godoc
// @Summary   Renew Akun Vless
// @Tags      Renew Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     username path string true "Username akun"
// @Param     expired formData int true "Durasi masa aktif (hari)"
// @Success   200 {object} model.SuccessResponse{data=model.RenewData} "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/renew/vless/{username} [patch]
func (ctrl *VlessController) Renew(c fiber.Ctx) error {
	username := strings.TrimSpace(strings.ToLower(c.Params("username")))
	if username == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(&model.ErrorResponse{
			Success: false,
			Message: "no",
			Errors: fiber.Map{
				"username": []string{"username tidak boleh kosong"},
			},
		})
	}

	var req model.RenewRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(&model.ErrorResponse{
			Success: false,
			Message: "body payload tidak valid",
		})
	}
	if errs := ctrl.validate.ValidateStruct(&req); errs != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(&model.ErrorResponse{
			Success: false,
			Message: "no",
			Errors:  errs,
		})
	}

	execCtx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	existing, err := ctrl.userRepository.FindByUsername(execCtx, username)
	if err != nil {
		slog.Error("Gagal mengecek username",
			slog.String("protocol", "vless"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pengecekan username gagal",
		})
	}
	if existing == nil || existing.Protocol != "vless" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(&model.ErrorResponse{
			Success: false,
			Message: "no",
			Errors: fiber.Map{
				"username": []string{"username tidak ditemukan"},
			},
		})
	}
	if existing.Status != "active" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(&model.ErrorResponse{
			Success: false,
			Message: "no",
			Errors: fiber.Map{
				"username": []string{"status user tidak aktif, tidak dapat diperpanjang"},
			},
		})
	}

	oldExpired := existing.ExpiredAt
	now := time.Now()
	baseTime := oldExpired
	if oldExpired.Before(now) {
		baseTime = now
	}
	newExpired := baseTime.AddDate(0, 0, req.Expired)

	if err := ctrl.userRepository.UpdateExpiredByUsername(execCtx, username, newExpired); err != nil {
		slog.Error("Gagal memperbarui masa aktif ke database",
			slog.String("protocol", "vless"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "gagal memperbarui masa aktif",
		})
	}

	return c.Status(fiber.StatusOK).JSON(&model.SuccessResponse{
		Success: true,
		Message: "ok",
		Data: model.RenewData{
			Username: existing.Username,
			From:     oldExpired.Format("2006-01-02 15:04:05"),
			To:       newExpired.Format("2006-01-02 15:04:05"),
		},
	})
}
