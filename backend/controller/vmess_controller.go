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

type VmessController struct {
	cfg            *config.Config
	validate       *validator.Validator
	vmessService   *service.VmessService
	userRepository *repository.UserRepository
}

func NewVmessController(cfg *config.Config, validate *validator.Validator, vmessService *service.VmessService, userRepository *repository.UserRepository) *VmessController {
	return &VmessController{
		cfg:            cfg,
		validate:       validate,
		vmessService:   vmessService,
		userRepository: userRepository,
	}
}

// TrialVmess godoc
// @Summary   Trial Akun Vmess
// @Tags      Trial Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     expired formData int true "Durasi Trial (menit)"
// @Success   200 {object} model.SuccessResponse{data=model.VmessData} "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/trial/vmess [post]
func (ctrl *VmessController) Trial(c fiber.Ctx) error {
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
	password := utils.RandomPassword("vmess")
	expired := time.Now().Add(time.Duration(req.Expired) * time.Minute)

	execCtx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	if err := ctrl.vmessService.AddUser(execCtx, username, password); err != nil {
		slog.Error("Gagal membuat trial akun",
			slog.String("protocol", "vmess"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pembuatan akun gagal",
		})
	}

	user := model.User{
		Protocol:   "vmess",
		Username:   username,
		Password:   password,
		LimitIP:    0,
		LimitQuota: 0,
		Status:     "active",
		ExpiredAt:  expired,
	}

	if err := ctrl.userRepository.Create(c.Context(), user); err != nil {
		slog.Error("Gagal menyimpan akun ke database",
			slog.String("protocol", "vmess"),
			slog.String("username", username),
			slog.Any("error", err))

		var rbErr error
		rbErr = ctrl.vmessService.DelUser(execCtx, username)
		if rbErr != nil {
			slog.Error("Rollback gagal, perlu pembersihan manual di sistem",
				slog.String("protocol", "vmess"),
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
		Data: model.VmessData{
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
				WS:      "vmess-ws",
				Upgrade: "vmess-up",
				GRPC:    "vmess-grpc",
			},
			Link: model.XrayLinkData{
				None:   service.GenerateVmessLink(username, hostname, 80, password, "ws", "vmess-ws", false),
				TLS:    service.GenerateVmessLink(username, hostname, 443, password, "ws", "vmess-ws", true),
				GRPC:   service.GenerateVmessLink(username, hostname, 443, password, "grpc", "vmess-grpc", true),
				UpNone: service.GenerateVmessLink(username, hostname, 80, password, "httpupgrade", "vmess-up", false),
				UpTLS:  service.GenerateVmessLink(username, hostname, 443, password, "httpupgrade", "vmess-up", true),
			},
		},
	})
}

// CreateVmess godoc
// @Summary   Create Akun Vmess
// @Tags      Create Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     username formData string true "Username akun"
// @Param     password formData string false "Pasword akun (akan dibuat random jika kosong)"
// @Param     limit_ip formData int false "Limit IP (unlimited jika kosong)"
// @Param     limit_quota formData int false "Limit Quota (unlimited jika kosong)"
// @Param     expired formData int true "Durasi masa aktif (hari)"
// @Success   200 {object} model.SuccessResponse{data=model.VmessData} "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/create/vmess [post]
func (ctrl *VmessController) Create(c fiber.Ctx) error {
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

	password := utils.RandomPassword("vmess")
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

	if err := ctrl.vmessService.AddUser(execCtx, username, password); err != nil {
		slog.Error("Gagal membuat akun",
			slog.String("protocol", "vmess"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pembuatan akun gagal",
		})
	}

	user := model.User{
		Protocol:   "vmess",
		Username:   username,
		Password:   password,
		LimitIP:    limitIP,
		LimitQuota: limitQuota,
		Status:     "active",
		ExpiredAt:  expired,
	}

	if err := ctrl.userRepository.Create(c.Context(), user); err != nil {
		slog.Error("Gagal menyimpan akun ke database",
			slog.String("protocol", "vmess"),
			slog.String("username", username),
			slog.Any("error", err))

		var rbErr error
		rbErr = ctrl.vmessService.DelUser(execCtx, username)
		if rbErr != nil {
			slog.Error("Rollback gagal, perlu pembersihan manual di sistem",
				slog.String("protocol", "vmess"),
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
		Data: model.VmessData{
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
				WS:      "vmess-ws",
				Upgrade: "vmess-up",
				GRPC:    "vmess-grpc",
			},
			Link: model.XrayLinkData{
				None:   service.GenerateVmessLink(username, hostname, 80, password, "ws", "vmess-ws", false),
				TLS:    service.GenerateVmessLink(username, hostname, 443, password, "ws", "vmess-ws", true),
				GRPC:   service.GenerateVmessLink(username, hostname, 443, password, "grpc", "vmess-grpc", true),
				UpNone: service.GenerateVmessLink(username, hostname, 80, password, "httpupgrade", "vmess-up", false),
				UpTLS:  service.GenerateVmessLink(username, hostname, 443, password, "httpupgrade", "vmess-up", true),
			},
		},
	})
}

// DeleteVmess godoc
// @Summary   Delete Akun Vmess
// @Tags      Delete Akun
// @Accept    json
// @Produce   json
// @Param     username path string true "Username akun"
// @Success   200 {object} model.SuccessResponse "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/delete/vmess/{username} [delete]
func (ctrl *VmessController) Delete(c fiber.Ctx) error {
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
			slog.String("protocol", "vmess"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pengecekan username gagal",
		})
	}
	if existing == nil || existing.Protocol != "vmess" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(&model.ErrorResponse{
			Success: false,
			Message: "no",
			Errors: fiber.Map{
				"username": []string{"username tidak ditemukan"},
			},
		})
	}

	if err := ctrl.vmessService.DelUser(execCtx, username); err != nil {
		slog.Error("Gagal menghapus user dari sistem",
			slog.String("protocol", "vmess"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "gagal menghapus user dari server",
		})
	}

	if err := ctrl.userRepository.DeleteByUsername(execCtx, username); err != nil {
		slog.Error("Gagal menghapus user dari database",
			slog.String("protocol", "vmess"),
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

// RenewVmess godoc
// @Summary   Renew Akun Vmess
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
// @Router    /vps/renew/vmess/{username} [patch]
func (ctrl *VmessController) Renew(c fiber.Ctx) error {
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
			slog.String("protocol", "vmess"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pengecekan username gagal",
		})
	}
	if existing == nil || existing.Protocol != "vmess" {
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
			slog.String("protocol", "vmess"),
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
