package db

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// closedPort returns a localhost port nothing listens on.
func closedPort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	_ = l.Close()
	return addr
}

func TestConnectRefusedGivesHintWithoutPassword(t *testing.T) {
	dsn := "root:s3cret@tcp(" + closedPort(t) + ")/tinh_tien_go?parseTime=True"

	start := time.Now()
	_, err := connect(dsn, 0, mysql.Open)
	if err == nil {
		t.Fatal("expected an error")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatalf("timeout 0 must not retry, took %s", time.Since(start))
	}
	msg := err.Error()
	if strings.Contains(msg, "s3cret") {
		t.Fatalf("password leaked in error: %s", msg)
	}
	for _, want := range []string{"root@127.0.0.1:", "/tinh_tien_go", "make db-up"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q should contain %q", msg, want)
		}
	}
}

func TestConnectRetriesUntilTheDatabaseIsUp(t *testing.T) {
	attempts := 0
	open := func(string) gorm.Dialector {
		attempts++
		if attempts < 3 {
			// Not up yet: an address nothing listens on.
			return mysql.Open("u:p@tcp(" + closedPort(t) + ")/x")
		}
		return sqlite.Open(":memory:")
	}

	db, err := connect("u:p@tcp(127.0.0.1:1)/x", 10*time.Second, open)
	if err != nil {
		t.Fatalf("expected success after retries, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
}

func TestDescribeDSN(t *testing.T) {
	got := describeDSN("root:pw@tcp(127.0.0.1:3307)/tinh_tien_go?charset=utf8mb4")
	if got != "root@127.0.0.1:3307/tinh_tien_go" {
		t.Fatalf("describeDSN = %q", got)
	}
	if describeDSN("::bad::") != "(invalid DSN)" && strings.Contains(describeDSN("::bad::"), "pw") {
		t.Fatal("invalid DSN must not echo input")
	}
}

func TestHintForAuthError(t *testing.T) {
	err := errors.New("Error 1045 (28000): Access denied")
	if !strings.Contains(hint(err), "DB_DSN") {
		t.Fatal("generic hint should mention DB_DSN")
	}
}
