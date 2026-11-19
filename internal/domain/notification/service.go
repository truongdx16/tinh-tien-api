package notification

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidToken = errors.New("device token is required")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// UpsertDevice stores token for userID (moving it from another account if
// the device signed in with a different one).
func (r *Repository) UpsertDevice(userID, token, platform string, now time.Time) error {
	var existing DeviceToken
	err := r.db.Unscoped().Where("token = ?", token).First(&existing).Error
	switch {
	case err == nil:
		return r.db.Unscoped().Model(&existing).Updates(map[string]any{
			"user_id":      userID,
			"platform":     platform,
			"last_seen_at": now,
			"deleted_at":   nil,
		}).Error
	case errors.Is(err, gorm.ErrRecordNotFound):
		return r.db.Create(&DeviceToken{UserID: userID, Token: token, Platform: platform, LastSeenAt: now}).Error
	default:
		return err
	}
}

// DeleteDevices removes tokens for good (no soft delete: they never come
// back).
func (r *Repository) DeleteDevices(tokens ...string) error {
	if len(tokens) == 0 {
		return nil
	}
	return r.db.Unscoped().Where("token IN ?", tokens).Delete(&DeviceToken{}).Error
}

// DeleteUserDevice removes token only when it belongs to userID.
func (r *Repository) DeleteUserDevice(userID, token string) error {
	return r.db.Unscoped().Where("token = ? AND user_id = ?", token, userID).Delete(&DeviceToken{}).Error
}

// ActiveTokens: devices of active accounts (a disabled account receives
// nothing). userID limits to one account when not empty.
func (r *Repository) ActiveTokens(userID string) ([]string, error) {
	var tokens []string
	q := r.db.Table("device_tokens").
		Joins("JOIN users ON users.id = device_tokens.user_id").
		Where("device_tokens.deleted_at IS NULL AND users.deleted_at IS NULL AND users.active = ?", true)
	if userID != "" {
		q = q.Where("device_tokens.user_id = ?", userID)
	}
	err := q.Pluck("device_tokens.token", &tokens).Error
	return tokens, err
}

// CreateOnce inserts n unless a notification with the same RefKey exists.
// Returns whether it was inserted.
func (r *Repository) CreateOnce(n *Notification) (bool, error) {
	res := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(n)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *Repository) SetSentCount(n *Notification, sent int) error {
	return r.db.Model(n).Update("sent_count", sent).Error
}

func (r *Repository) List(limit int) ([]Notification, error) {
	var items []Notification
	err := r.db.Order("created_at desc").Limit(limit).Find(&items).Error
	return items, err
}

type Service struct {
	repo   *Repository
	sender Sender
	now    func() time.Time
}

func NewService(repo *Repository, sender Sender) *Service {
	if sender == nil {
		sender = NoopSender{}
	}
	return &Service{repo: repo, sender: sender, now: time.Now}
}

// PushEnabled: Firebase credentials are configured.
func (s *Service) PushEnabled() bool { return s.sender.Enabled() }

func (s *Service) RegisterDevice(userID, token, platform string) error {
	token = strings.TrimSpace(token)
	if token == "" || len(token) > 255 {
		return ErrInvalidToken
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform != "android" && platform != "ios" {
		platform = ""
	}
	return s.repo.UpsertDevice(userID, token, platform, s.now())
}

func (s *Service) UnregisterDevice(userID, token string) error {
	return s.repo.DeleteUserDevice(userID, strings.TrimSpace(token))
}

func (s *Service) List(limit int) ([]Notification, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.List(limit)
}

// Notify stores n and pushes it to every active device, once per RefKey:
// returns false (nothing sent) when it was already notified.
func (s *Service) Notify(ctx context.Context, n Notification) (bool, error) {
	created, err := s.repo.CreateOnce(&n)
	if err != nil || !created {
		return false, err
	}
	tokens, err := s.repo.ActiveTokens("")
	if err != nil {
		return true, err
	}
	sent, err := s.push(ctx, tokens, n)
	if updateErr := s.repo.SetSentCount(&n, sent); updateErr != nil && err == nil {
		err = updateErr
	}
	return true, err
}

// SendTest pushes a test message to userID's devices only (not stored).
// Returns the number of devices reached.
func (s *Service) SendTest(ctx context.Context, userID string) (int, error) {
	tokens, err := s.repo.ActiveTokens(userID)
	if err != nil {
		return 0, err
	}
	return s.push(ctx, tokens, Notification{
		Kind:     KindTest,
		Title:    "Thông báo thử",
		Body:     "Điện thoại này đã nhận được thông báo từ Tính Tiền Rau.",
		Endpoint: "planting-schedules",
	})
}

func (s *Service) push(ctx context.Context, tokens []string, n Notification) (int, error) {
	if len(tokens) == 0 || !s.sender.Enabled() {
		return 0, nil
	}
	sent, invalid, err := s.sender.Send(ctx, tokens, Message{
		Title: n.Title,
		Body:  n.Body,
		Data:  map[string]string{"kind": n.Kind, "endpoint": n.Endpoint},
	})
	if len(invalid) > 0 {
		if delErr := s.repo.DeleteDevices(invalid...); delErr != nil {
			log.Printf("notification: delete invalid tokens: %v", delErr)
		}
	}
	return sent, err
}
