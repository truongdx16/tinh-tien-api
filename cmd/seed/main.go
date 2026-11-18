package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"tinh-tien-api/internal/app"
	"tinh-tien-api/internal/pkg/config"
	"tinh-tien-api/internal/pkg/db"
)

func main() {
	migrate := flag.Bool("migrate", false, "run database migrations before seeding")
	flag.Parse()

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

	if *migrate {
		if err := app.Migrate(gormDB); err != nil {
			log.Fatalf("migrate: %v", err)
		}
		log.Println("migrations completed")
	}

	opts := app.DefaultSeedOptions()
	if v := os.Getenv("OWNER_USERNAME"); v != "" {
		opts.OwnerUsername = v
	}
	if v := os.Getenv("OWNER_PHONE"); v != "" {
		opts.OwnerPhone = v
	}
	if v := os.Getenv("OWNER_FULL_NAME"); v != "" {
		opts.OwnerFullName = v
	}
	if v := os.Getenv("OWNER_PASSWORD"); v != "" {
		opts.OwnerPassword = v
	} else if os.Getenv("APP_ENV") == "production" {
		log.Fatal("seed: OWNER_PASSWORD is required when APP_ENV=production (default owner123 is not allowed)")
	}
	if err := app.Seed(gormDB, cfg, opts); err != nil {
		log.Fatalf("seed: %v", err)
	}

	fmt.Printf("seed completed (owner=%s)\n", opts.OwnerUsername)
}
