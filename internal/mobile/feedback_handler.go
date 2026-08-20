package mobile

import (
	"net/http"

	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/domain/feedback"
	"tinh-tien-api/internal/mobile/adapter"
	"tinh-tien-api/internal/pkg/httputil"
)

type MobileFeedbackDto struct {
	ID        string  `json:"id"`
	Content   string  `json:"content"`
	Rating    *int    `json:"rating"`
	CreatedAt *string `json:"created_at"`
	UserID    *string `json:"user_id"`
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
		adapter.Fail(w, http.StatusInternalServerError, "failed to list feedback")
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
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	userID := auth.UserIDFromContext(r.Context())
	fullName := ""
	if body.FullName != nil {
		fullName = *body.FullName
	}
	f, err := h.svc.Create(body.Content, body.Rating, fullName, userID)
	if err != nil {
		adapter.Fail(w, http.StatusBadRequest, err.Error())
		return
	}
	adapter.Created(w, "feedback created", toFeedbackDto(*f))
}

func toFeedbackDto(f feedback.Feedback) MobileFeedbackDto {
	id := f.ID.String()
	ca := f.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	uid := f.UserID
	var uidPtr *string
	if uid != "" {
		uidPtr = &uid
	}
	_ = id
	return MobileFeedbackDto{
		ID:        f.ID.String(),
		Content:   f.Content,
		Rating:    f.Rating,
		CreatedAt: &ca,
		UserID:    uidPtr,
	}
}
