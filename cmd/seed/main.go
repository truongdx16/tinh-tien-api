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
	if err := app.Seed(gormDB, cfg, opts); err != nil {
		log.Fatalf("seed: %v", err)
	}

	fmt.Printf("seed completed (owner=%s)\n", opts.OwnerUsername)
}
