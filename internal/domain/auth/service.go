package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserInactive       = errors.New("user is inactive")
	ErrUserNotFound       = errors.New("user not found")
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     Role   `json:"role"`
	jwt.RegisteredClaims
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByUsername(username string) (*User, error) {
	var user User
	err := r.db.Where("username = ?", username).Limit(1).Find(&user).Error
	if err != nil {
		return nil, err
	}
	if user.ID == uuid.Nil {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (r *Repository) FindByID(id string) (*User, error) {
	var user User
	err := r.db.First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *Repository) Create(user *User) error {
	return r.db.Create(user).Error
}

func (r *Repository) List(page, pageSize int) ([]User, int64, error) {
	var total int64
	if err := r.db.Model(&User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []User
	offset := (page - 1) * pageSize
	err := r.db.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&users).Error
	return users, total, err
}

func (r *Repository) Update(user *User) error {
	return r.db.Save(user).Error
}

func (r *Repository) UpdateLastLogin(userID string) error {
	now := time.Now()
	return r.db.Model(&User{}).Where("id = ?", userID).Update("last_login_at", now).Error
}

type Service struct {
	repo      *Repository
	jwtSecret []byte
	tokenTTL  time.Duration
}

func NewService(repo *Repository, jwtSecret string, tokenTTL time.Duration) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  tokenTTL,
	}
}

func (s *Service) Login(req LoginRequest) (*LoginResponse, error) {
	user, err := s.repo.FindByUsername(req.Username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if !user.Active {
		return nil, ErrUserInactive
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, expiresAt, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}
	_ = s.repo.UpdateLastLogin(user.ID.String())

	return &LoginResponse{
		Token:     token,
		ExpiresIn: int64(time.Until(expiresAt).Seconds()),
		User:      toUserResponse(user),
	}, nil
}

func (s *Service) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (s *Service) CreateUser(req CreateUserRequest) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	role := req.Role
	if role == "" {
		role = RoleStaff
	}
	user := &User{
		Username:     req.Username,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		Role:         role,
		Active:       true,
	}
	if err := s.repo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) ListUsers(page, pageSize int) ([]User, int64, error) {
	return s.repo.List(page, pageSize)
}

func (s *Service) GetUser(id string) (*User, error) {
	return s.repo.FindByID(id)
}

func (s *Service) UpdateUser(id string, req UpdateUserRequest) (*User, error) {
	user, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Active != nil {
		user.Active = *req.Active
	}
	if req.Password != nil && *req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hash)
	}
	if err := s.repo.Update(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) EnsureDefaultOwner(username, password, fullName string) error {
	_, err := s.repo.FindByUsername(username)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrUserNotFound) {
		return err
	}
	_, err = s.CreateUser(CreateUserRequest{
		Username: username,
		Password: password,
		FullName: fullName,
		Role:     RoleOwner,
	})
	return err
}

func (s *Service) generateToken(user *User) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.tokenTTL)
	claims := &Claims{
		UserID:   user.ID.String(),
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	return signed, expiresAt, err
}

func toUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:       u.ID.String(),
		Username: u.Username,
		FullName: u.FullName,
		Role:     u.Role,
	}
}
