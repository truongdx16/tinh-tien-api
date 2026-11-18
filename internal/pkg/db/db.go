package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens the MySQL/MariaDB connection, waiting for the server when it
// is still starting (e.g. right after `docker start`): it retries with
// backoff for DB_CONNECT_TIMEOUT (default 30s, e.g. "60s"; "0" = one try).
func Connect(dsn string) (*gorm.DB, error) {
	return connect(dsn, connectTimeout(), mysql.Open)
}

func connect(dsn string, timeout time.Duration, open func(string) gorm.Dialector) (*gorm.DB, error) {
	logLevel := logger.Warn
	if os.Getenv("DB_LOG") == "info" {
		logLevel = logger.Info
	}

	deadline := time.Now().Add(timeout)
	wait := 500 * time.Millisecond
	for attempt := 1; ; attempt++ {
		db, err := openAndPing(open(dsn), logLevel)
		if err == nil {
			if attempt > 1 {
				log.Printf("database %s ready after %d attempts", describeDSN(dsn), attempt)
			}
			return db, nil
		}
		if time.Now().Add(wait).After(deadline) {
			return nil, fmt.Errorf("connect database %s: %w\n%s", describeDSN(dsn), err, hint(err))
		}
		log.Printf("database %s not ready (%v), retrying in %s...", describeDSN(dsn), shortErr(err), wait)
		time.Sleep(wait)
		if wait *= 2; wait > 5*time.Second {
			wait = 5 * time.Second
		}
	}
}

func openAndPing(dialector gorm.Dialector, level logger.LogLevel) (*gorm.DB, error) {
	db, err := gorm.Open(dialector, &gorm.Config{Logger: newQuietLogger(level)})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	// Recycle connections before MariaDB's wait_timeout closes them, so a
	// long-idle API does not fail its first request with "invalid connection".
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)
	return db, nil
}

func connectTimeout() time.Duration {
	if v := os.Getenv("DB_CONNECT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if v == "0" {
			return 0
		}
		log.Printf("invalid DB_CONNECT_TIMEOUT %q, using 30s", v)
	}
	return 30 * time.Second
}

// describeDSN returns "user@host:port/db" without the password, for logs.
func describeDSN(dsn string) string {
	cfg, err := drivermysql.ParseDSN(dsn)
	if err != nil {
		return "(invalid DSN)"
	}
	return fmt.Sprintf("%s@%s/%s", cfg.User, cfg.Addr, cfg.DBName)
}

func shortErr(err error) string {
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 && strings.Contains(msg, "dial tcp") {
		return msg[i+2:]
	}
	return msg
}

// hint says what to do for the usual local failures.
func hint(err error) string {
	msg := err.Error()
	var myErr *drivermysql.MySQLError
	switch {
	case strings.Contains(msg, "connection refused"), strings.Contains(msg, "no such host"),
		strings.Contains(msg, "i/o timeout"):
		return "hint: the database is not running or DB_DSN has the wrong host/port. " +
			"Run `make db-up` (starts DB_CONTAINER, or docker compose) and check DB_DSN in .env."
	case errors.As(err, &myErr) && myErr.Number == 1045:
		return "hint: wrong user/password in DB_DSN."
	case errors.As(err, &myErr) && myErr.Number == 1049:
		return "hint: the database in DB_DSN does not exist; create it or fix the name."
	default:
		return "hint: check DB_DSN in .env and that the database server is reachable."
	}
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
