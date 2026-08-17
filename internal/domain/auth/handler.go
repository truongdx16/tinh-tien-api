package auth

import (
	"errors"
	"net/http"
	"strings"

	"tinh-tien-api/internal/pkg/httputil"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	resp, err := h.svc.Login(req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			httputil.Unauthorized(w, "login failed", err.Error())
		case errors.Is(err, ErrUserInactive):
			httputil.Forbidden(w, "account inactive", err.Error())
		default:
			httputil.InternalError(w, "login failed", err.Error())
		}
		return
	}
	httputil.OK(w, "login successful", resp)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	page := httputil.ParsePageParams(r)
	users, total, err := h.svc.ListUsers(page.Page, page.PageSize)
	if err != nil {
		httputil.InternalError(w, "failed to list users", err.Error())
		return
	}
	resp := make([]UserResponse, 0, len(users))
	for i := range users {
		resp = append(resp, toUserResponse(&users[i]))
	}
	httputil.OKWithPagination(w, "users retrieved", resp, httputil.NewPagination(page.Page, page.PageSize, total))
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	user, err := h.svc.CreateUser(req)
	if err != nil {
		httputil.Fail(w, http.StatusBadRequest, "failed to create user", err.Error())
		return
	}
	httputil.Created(w, "user created", toUserResponse(user))
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request, id string) {
	user, err := h.svc.GetUser(id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httputil.NotFound(w, "user not found")
			return
		}
		httputil.InternalError(w, "failed to get user", err.Error())
		return
	}
	httputil.OK(w, "user retrieved", toUserResponse(user))
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request, id string) {
	var req UpdateUserRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Fail(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	user, err := h.svc.UpdateUser(id, req)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			httputil.NotFound(w, "user not found")
			return
		}
		httputil.InternalError(w, "failed to update user", err.Error())
		return
	}
	httputil.OK(w, "user updated", toUserResponse(user))
}

func AuthMiddleware(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				httputil.Unauthorized(w, "unauthorized", "missing authorization header")
				return
			}
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				httputil.Unauthorized(w, "unauthorized", "invalid authorization header")
				return
			}
			claims, err := svc.ParseToken(parts[1])
			if err != nil {
				httputil.Unauthorized(w, "unauthorized", "invalid or expired token")
				return
			}
			ctx := WithContext(r.Context(), claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := FromContext(r.Context())
		if claims == nil || claims.Role != RoleOwner {
			httputil.Forbidden(w, "forbidden", "owner role required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
