package mobile

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

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
	Phone   *string `json:"phone"`
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
		adapter.Fail(w, http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ")
		return
	}
	// Flutter sends the "email" field: a username or a phone number.
	resp, err := h.svc.Login(auth.LoginRequest{Username: strings.TrimSpace(req.Email), Password: req.Password})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			adapter.Fail(w, http.StatusUnauthorized, "Sai tên đăng nhập, số điện thoại hoặc mật khẩu")
		case errors.Is(err, auth.ErrUserInactive):
			adapter.Fail(w, http.StatusForbidden, "Tài khoản đã bị khoá")
		default:
			adapter.Fail(w, http.StatusInternalServerError, "Đăng nhập thất bại, vui lòng thử lại")
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
		adapter.Fail(w, http.StatusUnauthorized, "Phiên đăng nhập đã hết hạn, vui lòng đăng nhập lại")
		return
	}
	user, err := h.svc.GetUser(claims.UserID)
	if err != nil {
		adapter.Fail(w, http.StatusNotFound, "Không tìm thấy người dùng")
		return
	}
	adapter.OK(w, "user retrieved", userToMobileDto(user))
}

// UpdateMe: PUT /auth/me {"name": "...", "phone": "..."} — the signed-in
// user's own name and sign-in phone. Omitted fields stay; phone "" removes it.
func (h *AuthMobileHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	claims := auth.FromContext(r.Context())
	if claims == nil {
		adapter.Fail(w, http.StatusUnauthorized, "Phiên đăng nhập đã hết hạn, vui lòng đăng nhập lại")
		return
	}
	var body struct {
		Name  *string `json:"name"`
		Phone *string `json:"phone"`
	}
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ")
		return
	}
	if body.Name != nil && strings.TrimSpace(*body.Name) == "" {
		adapter.Fail(w, http.StatusBadRequest, "Tên không được để trống")
		return
	}
	user, err := h.svc.UpdateProfile(claims.UserID, body.Name, body.Phone)
	switch {
	case err == nil:
		adapter.OK(w, "Đã cập nhật tài khoản", userToMobileDto(user))
	case errors.Is(err, auth.ErrInvalidPhone):
		adapter.Fail(w, http.StatusBadRequest, "Số điện thoại không hợp lệ")
	case errors.Is(err, auth.ErrPhoneTaken):
		adapter.Fail(w, http.StatusConflict, "Số điện thoại đã được dùng cho tài khoản khác")
	case errors.Is(err, auth.ErrUserNotFound):
		adapter.Fail(w, http.StatusNotFound, "Không tìm thấy người dùng")
	default:
		adapter.Fail(w, http.StatusInternalServerError, "Không cập nhật được tài khoản")
	}
}

// ChangePassword: POST /auth/change-password
// {"current_password": "...", "new_password": "..."} for the signed-in user.
func (h *AuthMobileHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := auth.FromContext(r.Context())
	if claims == nil {
		adapter.Fail(w, http.StatusUnauthorized, "Phiên đăng nhập đã hết hạn, vui lòng đăng nhập lại")
		return
	}
	var body struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := httputil.Decode(r, &body); err != nil {
		adapter.Fail(w, http.StatusBadRequest, "Dữ liệu gửi lên không hợp lệ")
		return
	}
	err := h.svc.ChangePassword(claims.UserID, body.CurrentPassword, body.NewPassword)
	switch {
	case err == nil:
		adapter.OK(w, "Đã đổi mật khẩu", nil)
	case errors.Is(err, auth.ErrWrongPassword):
		adapter.Fail(w, http.StatusBadRequest, "Mật khẩu hiện tại không đúng")
	case errors.Is(err, auth.ErrWeakPassword):
		adapter.Fail(w, http.StatusBadRequest,
			fmt.Sprintf("Mật khẩu mới cần ít nhất %d ký tự và khác mật khẩu hiện tại", auth.MinPasswordLength))
	case errors.Is(err, auth.ErrUserNotFound):
		adapter.Fail(w, http.StatusNotFound, "Không tìm thấy người dùng")
	default:
		adapter.Fail(w, http.StatusInternalServerError, "Không đổi được mật khẩu")
	}
}

func (h *AuthMobileHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, _, err := h.svc.ListUsers(1, 200)
	if err != nil {
		adapter.Fail(w, http.StatusInternalServerError, "Không tải được danh sách người dùng")
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
		Phone:   u.Phone,
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
		Phone:   u.Phone,
		StoreID: nil,
		Role:    string(u.Role),
		Status:  status,
	}
}
