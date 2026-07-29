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

type SSHController struct {
	cfg            *config.Config
	validate       *validator.Validator
	sshService     *service.SSHService
	userRepository *repository.UserRepository
}

func NewSSHController(cfg *config.Config, validate *validator.Validator, sshService *service.SSHService, userRepository *repository.UserRepository) *SSHController {
	return &SSHController{
		cfg:            cfg,
		validate:       validate,
		sshService:     sshService,
		userRepository: userRepository,
	}
}

// TrialSSH godoc
// @Summary   Trial Akun SSH
// @Tags      Trial Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     expired formData int true "Durasi Trial (menit)"
// @Success   200 {object} model.SuccessResponse{data=model.SSHData} "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/trial/ssh [post]
func (ctrl *SSHController) Trial(c fiber.Ctx) error {
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
	password := utils.RandomPassword("ssh")
	expired := time.Now().Add(time.Duration(req.Expired) * time.Minute)

	execCtx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	if err := ctrl.sshService.AddUser(execCtx, username, password); err != nil {
		slog.Error("Gagal membuat trial akun",
			slog.String("protocol", "ssh"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pembuatan akun gagal",
		})
	}

	user := model.User{
		Protocol:   "ssh",
		Username:   username,
		Password:   password,
		LimitIP:    0,
		LimitQuota: 0,
		Status:     "active",
		ExpiredAt:  expired,
	}

	if err := ctrl.userRepository.Create(c.Context(), user); err != nil {
		slog.Error("Gagal menyimpan akun ke database",
			slog.String("protocol", "ssh"),
			slog.String("username", username),
			slog.Any("error", err))

		var rbErr error
		rbErr = ctrl.sshService.DelUser(execCtx, username)
		if rbErr != nil {
			slog.Error("Rollback gagal, perlu pembersihan manual di sistem",
				slog.String("protocol", "ssh"),
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
		Data: model.SSHData{
			Hostname: hostname,
			ISP:      isp,
			City:     city,
			Username: username,
			Password: password,
			Expired:  expired.Format("2006-01-02 15:04:05"),
			Port: model.PortData{
				None:  []string{"80", "8080"},
				TLS:   []string{"443", "444", "8443"},
				UDPGW: []string{"7100", "7200", "7300", "7400", "7500", "7600"},
			},
			PayloadWS: model.PayloadWSData{
				PayloadCDN:      "GET / HTTP/1.1[crlf]Host: [host_port][crlf]User-Agent: [ua][crlf]Upgrade: websocket[crlf][crlf]",
				PayloadWithPath: "GET /worryfree/ssh HTTP/1.1[crlf]Host: BUG[crlf]User-Agent: [ua][crlf]Upgrade: websocket[crlf][crlf]",
			},
		},
	})
}

// CreateSSH godoc
// @Summary   Create Akun SSH
// @Tags      Create Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     username formData string true "Username akun"
// @Param     password formData string false "Pasword akun (akan dibuat random jika kosong)"
// @Param     limit_ip formData int false "Limit IP (unlimited jika kosong)"
// @Param     limit_quota formData int false "Limit Quota (unlimited jika kosong)"
// @Param     expired formData int true "Durasi masa aktif (hari)"
// @Success   200 {object} model.SuccessResponse{data=model.SSHData} "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/create/ssh [post]
func (ctrl *SSHController) Create(c fiber.Ctx) error {
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

	password := utils.RandomPassword("ssh")
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

	if err := ctrl.sshService.AddUser(execCtx, username, password); err != nil {
		slog.Error("Gagal membuat akun",
			slog.String("protocol", "ssh"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pembuatan akun gagal",
		})
	}

	user := model.User{
		Protocol:   "ssh",
		Username:   username,
		Password:   password,
		LimitIP:    limitIP,
		LimitQuota: limitQuota,
		Status:     "active",
		ExpiredAt:  expired,
	}

	if err := ctrl.userRepository.Create(c.Context(), user); err != nil {
		slog.Error("Gagal menyimpan akun ke database",
			slog.String("protocol", "ssh"),
			slog.String("username", username),
			slog.Any("error", err))

		var rbErr error
		rbErr = ctrl.sshService.DelUser(execCtx, username)
		if rbErr != nil {
			slog.Error("Rollback gagal, perlu pembersihan manual di sistem",
				slog.String("protocol", "ssh"),
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
		Data: model.SSHData{
			Hostname: hostname,
			ISP:      isp,
			City:     city,
			Username: username,
			Password: password,
			Expired:  expired.Format("2006-01-02 15:04:05"),
			Port: model.PortData{
				None:  []string{"80", "8080"},
				TLS:   []string{"443", "444", "8443"},
				UDPGW: []string{"7100", "7200", "7300", "7400", "7500", "7600"},
			},
			PayloadWS: model.PayloadWSData{
				PayloadCDN:      "GET / HTTP/1.1[crlf]Host: [host_port][crlf]User-Agent: [ua][crlf]Upgrade: websocket[crlf][crlf]",
				PayloadWithPath: "GET /worryfree/ssh HTTP/1.1[crlf]Host: BUG[crlf]User-Agent: [ua][crlf]Upgrade: websocket[crlf][crlf]",
			},
		},
	})
}

// DeleteSSH godoc
// @Summary   Delete Akun SSH
// @Tags      Delete Akun
// @Accept    json
// @Produce   json
// @Param     username path string true "Username akun"
// @Success   200 {object} model.SuccessResponse "ok"
// @Failure   400 {object} model.ErrorResponse "Bad Request"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/delete/ssh/{username} [delete]
func (ctrl *SSHController) Delete(c fiber.Ctx) error {
	username := strings.ToLower(strings.TrimSpace(c.Params("username")))
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
			slog.String("protocol", "ssh"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pengecekan username gagal",
		})
	}
	if existing == nil || existing.Protocol != "ssh" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(&model.ErrorResponse{
			Success: false,
			Message: "no",
			Errors: fiber.Map{
				"username": []string{"username tidak ditemukan"},
			},
		})
	}

	if err := ctrl.sshService.DelUser(execCtx, username); err != nil {
		slog.Error("Gagal menghapus user dari sistem",
			slog.String("protocol", "ssh"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "gagal menghapus user dari server",
		})
	}

	if err := ctrl.userRepository.DeleteByUsername(execCtx, username); err != nil {
		slog.Error("Gagal menghapus user dari database",
			slog.String("protocol", "ssh"),
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

// RenewSSH godoc
// @Summary   Renew Akun SSH
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
// @Router    /vps/renew/ssh/{username} [patch]
func (ctrl *SSHController) Renew(c fiber.Ctx) error {
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
			slog.String("protocol", "ssh"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pengecekan username gagal",
		})
	}
	if existing == nil || existing.Protocol != "ssh" {
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
			slog.String("protocol", "ssh"),
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

// DetailSSH godoc
// @Summary   Detail Akun SSH
// @Tags      Detail Akun
// @Accept    x-www-form-urlencoded
// @Produce   json
// @Param     username path string true "Username akun"
// @Success   200 {object} model.SuccessResponse{data=model.SSHData} "ok"
// @Failure   422 {object} model.ErrorResponse "Unprocessable Entity"
// @Failure   500 {object} model.ErrorResponse "Internal Server Error"
// @Security  BearerAuth
// @Router    /vps/detail/ssh/{username} [get]
func (ctrl *SSHController) Detail(c fiber.Ctx) error {
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
			slog.String("protocol", "ssh"),
			slog.String("username", username),
			slog.Any("error", err))
		return c.Status(fiber.StatusInternalServerError).JSON(&model.ErrorResponse{
			Success: false,
			Message: "pengecekan username gagal",
		})
	}
	if existing == nil || existing.Protocol != "ssh" {
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
		Data: model.SSHData{
			Hostname: hostname,
			ISP:      isp,
			City:     city,
			Username: existing.Username,
			Password: existing.Password,
			Expired:  existing.ExpiredAt.Format("2006-01-02 15:04:05"),
			Port: model.PortData{
				None:  []string{"80", "8080"},
				TLS:   []string{"443", "444", "8443"},
				UDPGW: []string{"7100", "7200", "7300", "7400", "7500", "7600"},
			},
			PayloadWS: model.PayloadWSData{
				PayloadCDN:      "GET / HTTP/1.1[crlf]Host: [host_port][crlf]User-Agent: [ua][crlf]Upgrade: websocket[crlf][crlf]",
				PayloadWithPath: "GET /worryfree/ssh HTTP/1.1[crlf]Host: BUG[crlf]User-Agent: [ua][crlf]Upgrade: websocket[crlf][crlf]",
			},
		},
	})
}

func ListSSH() {

}

func UpdatePasswordSSH() {

}

func UpdateLimitIP() {

}

func UpdateLimitQuota() {

}

func LockSSH() {

}

func UnlockSSH() {

}
