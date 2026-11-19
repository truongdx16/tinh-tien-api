package notification

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/planting"
	"tinh-tien-api/internal/pkg/model"
)

func openDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&auth.User{}, &DeviceToken{}, &Notification{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func addUser(t *testing.T, db *gorm.DB, name string, active bool) string {
	t.Helper()
	u := auth.User{Base: model.Base{ID: uuid.New()}, Username: name, PasswordHash: "x", FullName: name, Role: auth.RoleStaff, Active: true}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	if !active {
		db.Model(&u).Update("active", false)
	}
	return u.ID.String()
}

// fakeSender records pushes; tokens starting with "dead" are reported invalid.
type fakeSender struct {
	mu    sync.Mutex
	calls [][]string
	msgs  []Message
}

func (f *fakeSender) Enabled() bool { return true }

func (f *fakeSender) Send(_ context.Context, tokens []string, msg Message) (int, []string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	sorted := append([]string(nil), tokens...)
	sort.Strings(sorted)
	f.calls = append(f.calls, sorted)
	f.msgs = append(f.msgs, msg)
	var invalid []string
	for _, tk := range tokens {
		if strings.HasPrefix(tk, "dead") {
			invalid = append(invalid, tk)
		}
	}
	return len(tokens) - len(invalid), invalid, nil
}

func TestNotifyOncePerRefKeyToActiveDevices(t *testing.T) {
	db := openDB(t)
	sender := &fakeSender{}
	svc := NewService(NewRepository(db), sender)
	lan := addUser(t, db, "lan", true)
	hoa := addUser(t, db, "hoa", true)
	off := addUser(t, db, "off", false)
	for user, token := range map[string]string{lan: "tok-lan", hoa: "dead-hoa", off: "tok-off"} {
		if err := svc.RegisterDevice(user, token, "android"); err != nil {
			t.Fatal(err)
		}
	}

	n := Notification{Kind: KindHarvestToday, Title: "Đến ngày thu hoạch: Cải", RefKey: "harvest_today:x:2026-11-19", Endpoint: "planting-schedules/x"}
	isNew, err := svc.Notify(context.Background(), n)
	if err != nil || !isNew {
		t.Fatalf("first notify: new=%v err=%v", isNew, err)
	}
	if got := strings.Join(sender.calls[0], ","); got != "dead-hoa,tok-lan" {
		t.Fatalf("pushed to %s (disabled account must be skipped)", got)
	}
	if sender.msgs[0].Data["endpoint"] != "planting-schedules/x" || sender.msgs[0].Data["kind"] != KindHarvestToday {
		t.Errorf("data: %v", sender.msgs[0].Data)
	}

	// The dead token was removed.
	var count int64
	db.Model(&DeviceToken{}).Where("token = ?", "dead-hoa").Count(&count)
	if count != 0 {
		t.Error("invalid token kept")
	}

	// Same reminder again (next scheduler run): nothing new, nothing sent.
	isNew, err = svc.Notify(context.Background(), n)
	if err != nil || isNew || len(sender.calls) != 1 {
		t.Fatalf("second notify: new=%v err=%v calls=%d", isNew, err, len(sender.calls))
	}

	list, _ := svc.List(0)
	if len(list) != 1 || list[0].SentCount != 1 {
		t.Fatalf("stored: %+v", list)
	}
}

func TestRegisterDeviceMovesTokenBetweenAccounts(t *testing.T) {
	db := openDB(t)
	svc := NewService(NewRepository(db), &fakeSender{})
	a := addUser(t, db, "a", true)
	b := addUser(t, db, "b", true)

	if err := svc.RegisterDevice(a, " tok ", "ios"); err != nil {
		t.Fatal(err)
	}
	if err := svc.RegisterDevice(b, "tok", "ios"); err != nil {
		t.Fatal(err)
	}
	var rows []DeviceToken
	db.Find(&rows)
	if len(rows) != 1 || rows[0].UserID != b {
		t.Fatalf("rows: %+v", rows)
	}
	// A cannot unregister B's device; B can.
	_ = svc.UnregisterDevice(a, "tok")
	db.Find(&rows)
	if len(rows) != 1 {
		t.Fatal("another account removed the device")
	}
	_ = svc.UnregisterDevice(b, "tok")
	db.Find(&rows)
	if len(rows) != 0 {
		t.Fatal("device not removed")
	}
	if err := svc.RegisterDevice(a, "  ", ""); err != ErrInvalidToken {
		t.Fatalf("empty token: %v", err)
	}
}

func TestNotifyWithoutPushStillStores(t *testing.T) {
	db := openDB(t)
	svc := NewService(NewRepository(db), nil)
	isNew, err := svc.Notify(context.Background(), Notification{Kind: KindTest, Title: "t", RefKey: "k"})
	if err != nil || !isNew {
		t.Fatalf("new=%v err=%v", isNew, err)
	}
	if svc.PushEnabled() {
		t.Error("push reported enabled without credentials")
	}
}

type listSchedules []planting.PlantingSchedule

func (l listSchedules) List() ([]planting.PlantingSchedule, error) { return l, nil }

func TestSchedulerWaitsForTheReminderHour(t *testing.T) {
	db := openDB(t)
	sender := &fakeSender{}
	svc := NewService(NewRepository(db), sender)
	addUser(t, db, "lan", true)
	var user auth.User
	db.First(&user)
	_ = svc.RegisterDevice(user.ID.String(), "tok", "android")

	now := time.Date(2026, 11, 19, 6, 59, 0, 0, ict)
	sc := &Scheduler{
		Service:   svc,
		Schedules: listSchedules{schedule("Cải ngọt", "2026-11-19")},
		Location:  ict,
		Hour:      7,
		Now:       func() time.Time { return now },
	}
	if n, _ := sc.RunOnce(context.Background()); n != 0 {
		t.Fatalf("sent %d before 7:00", n)
	}
	now = now.Add(2 * time.Minute)
	if n, _ := sc.RunOnce(context.Background()); n != 1 {
		t.Fatalf("sent %d at 7:01, want 1", n)
	}
	now = now.Add(10 * time.Minute)
	if n, _ := sc.RunOnce(context.Background()); n != 0 {
		t.Fatalf("sent %d again on the next check", n)
	}
	if len(sender.calls) != 1 {
		t.Fatalf("pushes: %d", len(sender.calls))
	}
}

func TestFCMSender(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))

	tokenCalls := 0
	oauth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenCalls++
		_ = r.ParseForm()
		if r.Form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:jwt-bearer" || r.Form.Get("assertion") == "" {
			http.Error(w, "bad grant", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "ya29.test", "expires_in": 3600})
	}))
	defer oauth.Close()

	var payloads []fcmMessage
	fcm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ya29.test" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/v1/projects/demo/messages:send" {
			http.Error(w, "path "+r.URL.Path, http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var m fcmMessage
		_ = json.Unmarshal(body, &m)
		payloads = append(payloads, m)
		if m.Message.Token == "gone" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"code":404,"status":"NOT_FOUND","details":[{"errorCode":"UNREGISTERED"}]}}`))
			return
		}
		_, _ = w.Write([]byte(`{"name":"projects/demo/messages/1"}`))
	}))
	defer fcm.Close()

	raw, _ := json.Marshal(map[string]string{
		"project_id": "demo", "client_email": "svc@demo.iam.gserviceaccount.com",
		"private_key": pemKey, "token_uri": oauth.URL,
	})
	sa, err := ParseServiceAccount(raw)
	if err != nil {
		t.Fatal(err)
	}
	sender := NewFCMSender(sa)
	sender.BaseURL = fcm.URL

	msg := Message{Title: "Việc hôm nay", Body: "Phun thuốc", Data: map[string]string{"endpoint": "planting-schedules/1"}}
	sent, invalid, err := sender.Send(context.Background(), []string{"ok", "gone"}, msg)
	if err != nil || sent != 1 || len(invalid) != 1 || invalid[0] != "gone" {
		t.Fatalf("sent=%d invalid=%v err=%v", sent, invalid, err)
	}
	p := payloads[0].Message
	if p.Notification["title"] != "Việc hôm nay" || p.Data["endpoint"] != "planting-schedules/1" {
		t.Errorf("payload: %+v", p)
	}
	if p.Android["notification"].(map[string]any)["channel_id"] != "planting_schedule_channel" {
		t.Errorf("android channel: %+v", p.Android)
	}

	// The access token is reused.
	if _, _, err := sender.Send(context.Background(), []string{"ok"}, msg); err != nil {
		t.Fatal(err)
	}
	if tokenCalls != 1 {
		t.Errorf("oauth called %d times, want 1", tokenCalls)
	}
}
