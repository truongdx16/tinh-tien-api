package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tinh-tien-api/internal/app"
	"tinh-tien-api/internal/pkg/config"
	"tinh-tien-api/internal/pkg/db"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	gormDB, err := db.Connect(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	if err := app.Migrate(gormDB); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	handlers := app.BuildHandlers(gormDB, cfg)

	router := app.NewRouter(handlers)
	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Bind first so "listening" is only logged once the port is ours, and a
	// busy port fails with a clear message instead of after the log line.
	ln, err := net.Listen("tcp", cfg.Server.Addr)
	if err != nil {
		if errors.Is(err, syscall.EADDRINUSE) {
			log.Fatalf("listen %s: port already in use (another API instance?).\n"+
				"hint: run `make stop` to stop it, or set HTTP_ADDR to another port, e.g. HTTP_ADDR=:2190 make run", cfg.Server.Addr)
		}
		log.Fatalf("listen %s: %v", cfg.Server.Addr, err)
	}

	go func() {
		docsHost := cfg.Server.Addr
		if strings.HasPrefix(docsHost, ":") {
			docsHost = "localhost" + docsHost
		}
		log.Printf("server listening on %s (docs: http://%s/docs)", cfg.Server.Addr, docsHost)
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	fmt.Println("server stopped")
}
