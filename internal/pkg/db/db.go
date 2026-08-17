package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	logLevel := logger.Warn
	if os.Getenv("DB_LOG") == "info" {
		logLevel = logger.Info
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newQuietLogger(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)

	return db, nil
}

// quietLogger skips logging ErrRecordNotFound (normal "does not exist yet" lookups).
type quietLogger struct {
	level logger.LogLevel
}

func newQuietLogger(level logger.LogLevel) logger.Interface {
	return &quietLogger{level: level}
}

func (l *quietLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &quietLogger{level: level}
}

func (l *quietLogger) Info(_ context.Context, msg string, data ...any) {
	if l.level >= logger.Info {
		log.Printf(msg, data...)
	}
}

func (l *quietLogger) Warn(_ context.Context, msg string, data ...any) {
	if l.level >= logger.Warn {
		log.Printf(msg, data...)
	}
}

func (l *quietLogger) Error(_ context.Context, msg string, data ...any) {
	if l.level >= logger.Error {
		log.Printf(msg, data...)
	}
}

func (l *quietLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.level <= logger.Silent {
		return
	}
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}
	logger.Default.LogMode(l.level).Trace(context.Background(), begin, fc, err)
}
