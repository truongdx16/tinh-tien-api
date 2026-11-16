package mobile

import (
	"net/http"
	"strings"
	"time"

	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/feedback"
	"tinh-tien-api/internal/mobile/adapter"
	"tinh-tien-api/internal/pkg/httputil"
)

// MobileFeedbackDto maps to Flutter FeedbackNote (id and user_id are int there).
type MobileFeedbackDto struct {
	ID        uint    `json:"id"`
	Content   string  `json:"content"`
	Rating    *int    `json:"rating"`
	CreatedAt *string `json:"created_at"`
	UserID    *int    `json:"user_id"` // users use UUIDs → always null
}

type FeedbackMobileHandler struct {
	svc *feedback.Service
}

func NewFeedbackMobileHandler(svc *feedback.Service) *FeedbackMobileHandler {
	return &FeedbackMobileHandler{svc: svc}
}

func (h *FeedbackMobileHandler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.List()
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "Không tải được danh sách phản hồi")
		return
	}
	dtos := make([]MobileFeedbackDto, 0, len(items))
	for _, f := range items {
		dtos = append(dtos, toFeedbackDto(f))
	}
	adapter.OK(w, "feedback retrieved", dtos)
}

func (h *FeedbackMobileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Content  string  `json:"content"`
		Rating   *int    `json:"rating"`
		FullName *string `json:"full_name"`
	}
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ")
		return
	}
	if strings.TrimSpace(body.Content) == "" {
		adapter.Fail(w, http.StatusBadRequest, "Vui lòng nhập nội dung phản hồi")
		return
	}
	userID := auth.UserIDFromContext(r.Context())
	fullName := ""
	if body.FullName != nil {
		fullName = *body.FullName
	}
	f, err := h.svc.Create(body.Content, body.Rating, fullName, userID)
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, userMessage(err, "Không gửi được phản hồi"))
		return
	}
	adapter.Created(w, "feedback created", toFeedbackDto(*f))
}

func toFeedbackDto(f feedback.Feedback) MobileFeedbackDto {
	ca := f.CreatedAt.Format(time.RFC3339)
	return MobileFeedbackDto{
		ID:        f.ID,
		Content:   f.Content,
		Rating:    f.Rating,
		CreatedAt: &ca,
	}
}
