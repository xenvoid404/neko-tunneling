package utils

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/proxy/trojan"
	"github.com/xtls/xray-core/proxy/vless"
	"github.com/xtls/xray-core/proxy/vmess"
)

func getTagsByProtocol(targetProtocol string) []string {
	allTags := []string{
		"vmess-ws", "vmess-grpc", "vmess-up",
		"vless-ws", "vless-grpc", "vless-up",
		"trojan-ws", "trojan-grpc", "trojan-up",
	}

	var filtered []string
	prefix := targetProtocol + "-"
	for _, tag := range allTags {
		if strings.HasPrefix(tag, prefix) {
			filtered = append(filtered, tag)
		}
	}
	return filtered
}

func buildAccountMsg(protoType, uuid string) (*anypb.Any, error) {
	switch protoType {
	case "vless":
		return serial.ToTypedMessage(&vless.Account{Id: uuid}), nil
	case "vmess":
		return serial.ToTypedMessage(&vmess.Account{Id: uuid, AlterId: 0}), nil
	case "trojan":
		return serial.ToTypedMessage(&trojan.Account{Password: uuid}), nil
	default:
		return nil, fmt.Errorf("protokol %s tidak didukung", protoType)
	}
}

func AddXrayUser(apiAddr, targetProtocol, uuid, username string) error {
	targetTags := getTagsByProtocol(targetProtocol)
	if len(targetTags) == 0 {
		return fmt.Errorf("tidak ditemukan inbound tag untuk protokol: %s", targetProtocol)
	}

	conn, err := grpc.Dial(apiAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("gagal terhubung ke API Xray: %w", err)
	}
	defer conn.Close()
	client := command.NewHandlerServiceClient(conn)

	accountMsg, err := buildAccountMsg(targetProtocol, uuid)
	if err != nil {
		return fmt.Errorf("gagal memformat akun: %w", err)
	}

	user := &protocol.User{
		Level:   0,
		Email:   username,
		Account: accountMsg,
	}
	operation, _ := serial.ToTypedMessage(&command.AddUserOperation{User: user})

	for _, tag := range targetTags {
		req := &command.AlterInboundRequest{
			Tag:       tag,
			Operation: operation,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err := client.AlterInbound(ctx, req)
		defer cancel()

		return nil
	}
}
