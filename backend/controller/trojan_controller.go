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

type TrojanController struct {
	cfg            *config.Config
	validate       *validator.Validator
	trojanService  *service.TrojanService
	userRepository *repository.UserRepository
}

func NewTrojanController(cfg *config.Config, validate *validator.Validator, trojanService *service.TrojanService, userRepository *repository.UserRepository) *TrojanController {
	return &TrojanController{
		cfg:            cfg,
		validate:       validate,
		trojanService:  trojanService,
		userRepository: userRepository,
	}
}

// TrialTrojan godoc
// @Summary   Trial Akun Trojan
// @Tags      Trial Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     expired formData int true "Durasi Trial (menit)"
// @Success   200 {object} model.SuccessResponse{data=model.TrojanData} "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/trial/trojan [post]
func (ctrl *TrojanController) Trial(c fiber.Ctx) error {
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
	password := utils.RandomPassword("trojan")
	expired := time.Now().Add(time.Duration(req.Expired) * time.Minute)

	execCtx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	if err := ctrl.trojanService.AddUser(execCtx, username, password); err != nil {
		slog.Error("Gagal membuat trial akun",
			slog.String("protocol", "trojan"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pembuatan akun gagal",
		})
	}

	user := model.User{
		Protocol:   "trojan",
		Username:   username,
		Password:   password,
		LimitIP:    0,
		LimitQuota: 0,
		Status:     "active",
		ExpiredAt:  expired,
	}

	if err := ctrl.userRepository.Create(c.Context(), user); err != nil {
		slog.Error("Gagal menyimpan akun ke database",
			slog.String("protocol", "trojan"),
			slog.String("username", username),
			slog.Any("error", err))

		var rbErr error
		rbErr = ctrl.trojanService.DelUser(execCtx, username)
		if rbErr != nil {
			slog.Error("Rollback gagal, perlu pembersihan manual di sistem",
				slog.String("protocol", "trojan"),
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
		Data: model.TrojanData{
			Hostname: hostname,
			ISP:      isp,
			City:     city,
			Username: username,
			Password: password,
			Expired:  expired.Format("2006-01-02 15:04:05"),
			Port: model.PortData{
				TLS:  []string{"443"},
				GRPC: []string{"443"},
			},
			Path: model.PathData{
				WS:      "trojan-ws",
				Upgrade: "trojan-up",
				GRPC:    "trojan-grpc",
			},
			Link: model.XrayLinkData{
				TLS:   service.GenerateTrojanLink(username, hostname, 443, password, "ws", "trojan-ws", true),
				GRPC:  service.GenerateTrojanLink(username, hostname, 443, password, "grpc", "trojan-grpc", true),
				UpTLS: service.GenerateTrojanLink(username, hostname, 443, password, "httpupgrade", "trojan-up", true),
			},
		},
	})
}

// CreateTrojan godoc
// @Summary   Create Akun Trojan
// @Tags      Create Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     username formData string true "Username akun"
// @Param     password formData string false "Pasword akun (akan dibuat random jika kosong)"
// @Param     limit_ip formData int false "Limit IP (unlimited jika kosong)"
// @Param     limit_quota formData int false "Limit Quota (unlimited jika kosong)"
// @Param     expired formData int true "Durasi masa aktif (hari)"
// @Success   200 {object} model.SuccessResponse{data=model.TrojanData} "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/create/trojan [post]
func (ctrl *TrojanController) Create(c fiber.Ctx) error {
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

	password := utils.RandomPassword("trojan")
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

	if err := ctrl.trojanService.AddUser(execCtx, username, password); err != nil {
		slog.Error("Gagal membuat akun",
			slog.String("protocol", "trojan"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pembuatan akun gagal",
		})
	}

	user := model.User{
		Protocol:   "trojan",
		Username:   username,
		Password:   password,
		LimitIP:    limitIP,
		LimitQuota: limitQuota,
		Status:     "active",
		ExpiredAt:  expired,
	}

	if err := ctrl.userRepository.Create(c.Context(), user); err != nil {
		slog.Error("Gagal menyimpan akun ke database",
			slog.String("protocol", "trojan"),
			slog.String("username", username),
			slog.Any("error", err))

		var rbErr error
		rbErr = ctrl.trojanService.DelUser(execCtx, username)
		if rbErr != nil {
			slog.Error("Rollback gagal, perlu pembersihan manual di sistem",
				slog.String("protocol", "trojan"),
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
		Data: model.TrojanData{
			Hostname: hostname,
			ISP:      isp,
			City:     city,
			Username: username,
			Password: password,
			Expired:  expired.Format("2006-01-02 15:04:05"),
			Port: model.PortData{
				TLS:  []string{"443"},
				GRPC: []string{"443"},
			},
			Path: model.PathData{
				WS:      "trojan-ws",
				Upgrade: "trojan-up",
				GRPC:    "trojan-grpc",
			},
			Link: model.XrayLinkData{
				TLS:   service.GenerateTrojanLink(username, hostname, 443, password, "ws", "trojan-ws", true),
				GRPC:  service.GenerateTrojanLink(username, hostname, 443, password, "grpc", "trojan-grpc", true),
				UpTLS: service.GenerateTrojanLink(username, hostname, 443, password, "httpupgrade", "trojan-up", true),
			},
		},
	})
}

// DeleteTrojan godoc
// @Summary   Delete Akun Trojan
// @Tags      Delete Akun
// @Accept    json
// @Produce   json
// @Param     username path string true "Username akun"
// @Success   200 {object} model.SuccessResponse "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/delete/trojan/{username} [delete]
func (ctrl *TrojanController) Delete(c fiber.Ctx) error {
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
			slog.String("protocol", "trojan"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pengecekan username gagal",
		})
	}
	if existing == nil || existing.Protocol != "trojan" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(&model.ErrorResponse{
			Success: false,
			Message: "no",
			Errors: fiber.Map{
				"username": []string{"username tidak ditemukan"},
			},
		})
	}

	if err := ctrl.trojanService.DelUser(execCtx, username); err != nil {
		slog.Error("Gagal menghapus user dari sistem",
			slog.String("protocol", "trojan"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "gagal menghapus user dari server",
		})
	}

	if err := ctrl.userRepository.DeleteByUsername(execCtx, username); err != nil {
		slog.Error("Gagal menghapus user dari database",
			slog.String("protocol", "trojan"),
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

// RenewTrojan godoc
// @Summary   Renew Akun Trojan
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
// @Router    /vps/renew/trojan/{username} [patch]
func (ctrl *TrojanController) Renew(c fiber.Ctx) error {
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
			slog.String("protocol", "trojan"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pengecekan username gagal",
		})
	}
	if existing == nil || existing.Protocol != "trojan" {
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
			slog.String("protocol", "trojan"),
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

// DetailTrojan godoc
// @Summary   Detail Akun Trojan
// @Tags      Detail Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     username path string true "Username akun"
// @Success   200 {object} model.SuccessResponse{data=model.TrojanData} "ok"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/detail/trojan/{username} [get]
func (ctrl *TrojanController) Detail(c fiber.Ctx) error {
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

	execCtx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	existing, err := ctrl.userRepository.FindByUsername(execCtx, username)
	if err != nil {
		slog.Error("Gagal mengecek username",
			slog.String("protocol", "trojan"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pengecekan username gagal",
		})
	}
	if existing == nil || existing.Protocol != "trojan" {
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
				"username": []string{"status user tidak aktif"},
			},
		})
	}

	hostname := utils.ReadFile(ctrl.cfg.CacheDomainPath)
	isp := utils.ReadFile(ctrl.cfg.CacheISPPath)
	city := utils.ReadFile(ctrl.cfg.CacheCityPath)

	return c.Status(fiber.StatusOK).JSON(&model.SuccessResponse{
		Success: true,
		Message: "ok",
		Data: model.TrojanData{
			Hostname: hostname,
			ISP:      isp,
			City:     city,
			Username: existing.Username,
			Password: existing.Password,
			Expired:  existing.ExpiredAt.Format("2006-01-02 15:04:05"),
			Port: model.PortData{
				TLS:  []string{"443"},
				GRPC: []string{"443"},
			},
			Path: model.PathData{
				WS:      "trojan-ws",
				Upgrade: "trojan-up",
				GRPC:    "trojan-grpc",
			},
			Link: model.XrayLinkData{
				TLS:   service.GenerateTrojanLink(existing.Username, hostname, 443, existing.Password, "ws", "trojan-ws", true),
				GRPC:  service.GenerateTrojanLink(existing.Username, hostname, 443, existing.Password, "grpc", "trojan-grpc", true),
				UpTLS: service.GenerateTrojanLink(existing.Username, hostname, 443, existing.Password, "httpupgrade", "trojan-up", true),
			},
		},
	})
}
