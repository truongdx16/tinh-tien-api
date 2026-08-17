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
	status := flag.Bool("status", false, "show migration status without applying")
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

	if *status {
		lines, err := app.MigrateStatus(gormDB)
		if err != nil {
			log.Fatalf("status: %v", err)
		}
		for _, line := range lines {
			fmt.Println(line)
		}
		return
	}

	if err := app.Migrate(gormDB); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	fmt.Println("migrations completed")
}
