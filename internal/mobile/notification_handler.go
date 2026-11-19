package mobile

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/notification"
	"tinh-tien-api/internal/mobile/adapter"
	"tinh-tien-api/internal/pkg/httputil"
)

type NotificationMobileHandler struct {
	svc *notification.Service
}

func NewNotificationMobileHandler(svc *notification.Service) *NotificationMobileHandler {
	return &NotificationMobileHandler{svc: svc}
}

// MobileNotificationDto is one entry of the notification list.
type MobileNotificationDto struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Endpoint  string `json:"endpoint"`
	CreatedAt string `json:"created_at"`
}

// RegisterDevice: POST /devices {"token": "<fcm token>", "platform": "android|ios"}
// for the signed-in user. Called after sign-in and when FCM rotates the token.
func (h *NotificationMobileHandler) RegisterDevice(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if userID == "" {
		adapter.Fail(w, http.StatusUnauthorized, "Phiên đăng nhập đã hết hạn, vui lòng đăng nhập lại")
		return
	}
	var body struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ")
		return
	}
	if err := h.svc.RegisterDevice(userID, body.Token, body.Platform); err != nil {
		if errors.Is(err, notification.ErrInvalidToken) {
			adapter.Fail(w, http.StatusBadRequest, "Thiếu mã thiết bị nhận thông báo")
			return
		}
		adapter.Fail(w, http.StatusInternalServerError, "Không đăng ký được thiết bị nhận thông báo")
		return
	}
	adapter.OK(w, "Đã đăng ký nhận thông báo", map[string]bool{"push_enabled": h.svc.PushEnabled()})
}

// UnregisterDevice: DELETE /devices/{token}. Called on sign-out and when the
// user turns notifications off.
func (h *NotificationMobileHandler) UnregisterDevice(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	if err := h.svc.UnregisterDevice(userID, chi.URLParam(r, "token")); err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "Không huỷ được đăng ký thông báo")
		return
	}
	adapter.OK(w, "Đã huỷ đăng ký nhận thông báo", nil)
}

// List: GET /notifications?limit=50, newest first.
func (h *NotificationMobileHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.svc.List(limit)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "Không tải được thông báo")
		return
	}
	dtos := make([]MobileNotificationDto, 0, len(items))
	for _, n := range items {
		dtos = append(dtos, MobileNotificationDto{
			ID:        n.ID.String(),
			Kind:      n.Kind,
			Title:     n.Title,
			Body:      n.Body,
			Endpoint:  n.Endpoint,
			CreatedAt: n.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	adapter.OK(w, "notifications retrieved", dtos)
}

// SendTest: POST /notifications/test pushes a test message to the caller's
// own devices, to check the setup end to end.
func (h *NotificationMobileHandler) SendTest(w http.ResponseWriter, r *http.Request) {
	if !h.svc.PushEnabled() {
		adapter.Fail(w, http.StatusServiceUnavailable, "Máy chủ chưa cấu hình gửi thông báo (Firebase)")
		return
	}
	sent, err := h.svc.SendTest(r.Context(), auth.UserIDFromContext(r.Context()))
	if err != nil && sent == 0 {
		adapter.Fail(w, http.StatusBadGateway, "Không gửi được thông báo thử")
		return
	}
	if sent == 0 {
		adapter.Fail(w, http.StatusNotFound, "Tài khoản chưa có thiết bị nào đăng ký nhận thông báo")
		return
	}
	adapter.OK(w, "Đã gửi thông báo thử", map[string]int{"sent": sent})
}
