package provision

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/proxy/trojan"
	"github.com/xtls/xray-core/proxy/vless"
	"github.com/xtls/xray-core/proxy/vmess"
)

var (
	xrayOnce    sync.Once
	xrayConn    *grpc.ClientConn
	xrayClient  command.HandlerServiceClient
	xrayInitErr error
)

var protocolTags = map[string][]string{
	"vmess":  {"vmess-ws", "vmess-grpc", "vmess-up"},
	"vless":  {"vless-ws", "vless-grpc", "vless-up"},
	"trojan": {"trojan-ws", "trojan-grpc", "trojan-up"},
}

func InitXrayClient(addr string) error {
	xrayOnce.Do(func() {
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			xrayInitErr = fmt.Errorf("gagal terhubung ke API Xray: %w", err)
			return
		}
		xrayConn = conn
		xrayClient = command.NewHandlerServiceClient(conn)
	})
	return xrayInitErr
}

func CloseXrayClient() error {
	if xrayConn == nil {
		return nil
	}
	return xrayConn.Close()
}

func buildAccountMsg(targetProtocol, secret string) (*serial.TypedMessage, error) {
	switch targetProtocol {
	case "vless":
		return serial.ToTypedMessage(&vless.Account{Id: secret}), nil
	case "vmess":
		return serial.ToTypedMessage(&vmess.Account{Id: secret}), nil
	case "trojan":
		return serial.ToTypedMessage(&trojan.Account{Password: secret}), nil
	default:
		return nil, fmt.Errorf("protokol %s tidak didukung", targetProtocol)
	}
}

func AddXrayUser(ctx context.Context, targetProtocol, secret, username string) error {
	if xrayClient == nil {
		return fmt.Errorf("xray client belum diinisialisasi, panggil InitXrayClient saat startup")
	}

	tags, ok := protocolTags[targetProtocol]
	if !ok {
		return fmt.Errorf("tidak ditemukan inbound tag untuk protokol: %s", targetProtocol)
	}

	accountMsg, err := buildAccountMsg(targetProtocol, secret)
	if err != nil {
		return fmt.Errorf("gagal memformat akun: %w", err)
	}

	operation := serial.ToTypedMessage(&command.AddUserOperation{
		User: &protocol.User{
			Level:   0,
			Email:   username,
			Account: accountMsg,
		},
	})

	applied := make([]string, 0, len(tags))
	for _, tag := range tags {
		callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		_, err := xrayClient.AlterInbound(callCtx, &command.AlterInboundRequest{
			Tag:       tag,
			Operation: operation,
		})
		cancel()

		if err != nil {
			if rbErr := removeFromTags(ctx, username, applied); rbErr != nil {
				return fmt.Errorf("gagal menambahkan user ke inbound %s: %w (rollback tag sebelumnya juga gagal: %v)", tag, err, rbErr)
			}
			return fmt.Errorf("gagal menambahkan user ke inbound %s: %w", tag, err)
		}
		applied = append(applied, tag)
	}

	return nil
}

func RemoveXrayUser(ctx context.Context, targetProtocol, username string) error {
	tags, ok := protocolTags[targetProtocol]
	if !ok {
		return fmt.Errorf("tidak ditemukan inbound tag untuk protokol: %s", targetProtocol)
	}
	return removeFromTags(ctx, username, tags)
}

func removeFromTags(ctx context.Context, username string, tags []string) error {
	if xrayClient == nil {
		return fmt.Errorf("xray client belum diinisialisasi, panggil InitXrayClient saat startup")
	}
	if len(tags) == 0 {
		return nil
	}

	operation := serial.ToTypedMessage(&command.RemoveUserOperation{Email: username})

	var failed []string
	for _, tag := range tags {
		callCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		_, err := xrayClient.AlterInbound(callCtx, &command.AlterInboundRequest{
			Tag:       tag,
			Operation: operation,
		})
		cancel()
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", tag, err))
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("gagal menghapus user dari sebagian inbound: %s", strings.Join(failed, "; "))
	}
	return nil
}
