package mobile

import (
	"errors"
	"net/http"

	"tinh-tien-api/internal/domain/auth"
	"tinh-tien-api/internal/mobile/adapter"
	"tinh-tien-api/internal/pkg/httputil"
)

// MobileLoginRequest is what Flutter sends: email (used as username), password, device_name.
type MobileLoginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	DeviceName string `json:"device_name"`
}

// MobileUserDto maps to Flutter UserDto.
type MobileUserDto struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Email   string  `json:"email"`
	StoreID *string `json:"store_id"`
	Role    string  `json:"role"`
	Status  int     `json:"status"`
}

// MobileLoginResponse maps to Flutter LoginResponseDto.
type MobileLoginResponse struct {
	Token string        `json:"token"`
	User  MobileUserDto `json:"user"`
}

type AuthMobileHandler struct {
	svc *auth.Service
}

func NewAuthMobileHandler(svc *auth.Service) *AuthMobileHandler {
	return &AuthMobileHandler{svc: svc}
}

func (h *AuthMobileHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req MobileLoginRequest
	if err := httputil.Decode(r, &req); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Flutter sends email field; we use it as username.
	resp, err := h.svc.Login(auth.LoginRequest{Username: req.Email, Password: req.Password})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			adapter.Fail(w, http.StatusUnauthorized, "invalid credentials")
		case errors.Is(err, auth.ErrUserInactive):
			adapter.Fail(w, http.StatusForbidden, "account inactive")
		default:
			adapter.Fail(w, http.StatusInternalServerError, "login failed")
		}
		return
	}
	adapter.OK(w, "login successful", MobileLoginResponse{
		Token: resp.Token,
		User:  toMobileUserDto(resp.User),
	})
}

func (h *AuthMobileHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Stateless JWT — no server-side invalidation; Flutter clears token locally.
	adapter.OK(w, "logged out", nil)
}

func (h *AuthMobileHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := auth.FromContext(r.Context())
	if claims == nil {
		adapter.Fail(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	user, err := h.svc.GetUser(claims.UserID)
	if err != nil {
		adapter.Fail(w, http.StatusNotFound, "user not found")
		return
	}
	adapter.OK(w, "user retrieved", userToMobileDto(user))
}

func (h *AuthMobileHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, _, err := h.svc.ListUsers(1, 200)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	dtos := make([]MobileUserDto, 0, len(users))
	for i := range users {
		dtos = append(dtos, userToMobileDto(&users[i]))
	}
	adapter.OK(w, "users retrieved", dtos)
}

func toMobileUserDto(u auth.UserResponse) MobileUserDto {
	return MobileUserDto{
		ID:      u.ID,
		Name:    u.FullName,
		Email:   u.Username,
		StoreID: nil,
		Role:    string(u.Role),
		Status:  1,
	}
}

func userToMobileDto(u *auth.User) MobileUserDto {
	status := adapter.BoolToStatus(u.Active)
	return MobileUserDto{
		ID:      u.ID.String(),
		Name:    u.FullName,
		Email:   u.Username,
		StoreID: nil,
		Role:    string(u.Role),
		Status:  status,
	}
}
