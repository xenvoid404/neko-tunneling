package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

var log = createLogger()
var cfg = getConfig()

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("Memulai layanan Multiplexer...")

	server, err := newServer()
	if err != nil {
		log.Error("Gagal menginisiasi Multiplexer",
			slog.Any("error", err))
		os.Exit(1)
	}

	go func() {
		if err := server.start(ctx); err != nil {
			log.Error("Multiplexer berhenti karena error",
				slog.Any("error", err))
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("Menerima sinyal shutdown, mematikan Multiplexer...")
}
