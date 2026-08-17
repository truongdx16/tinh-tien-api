package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	go func() {
		log.Printf("server listening on %s (docs: http://localhost%s/docs)", cfg.Server.Addr, cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
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
