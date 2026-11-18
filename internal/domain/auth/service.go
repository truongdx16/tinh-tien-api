package auth

import (
	"errors"
	"fmt"
	"strings"
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
	// ErrWrongPassword: the current password given to ChangePassword is wrong.
	ErrWrongPassword = errors.New("current password is wrong")
	// ErrWeakPassword: the new password is too short or equals the old one.
	ErrWeakPassword = errors.New("new password is too weak")
	// ErrInvalidPhone: not a Vietnamese phone number.
	ErrInvalidPhone = errors.New("invalid phone number")
	// ErrPhoneTaken: another account already signs in with this phone.
	ErrPhoneTaken = errors.New("phone number already used")
)

// MinPasswordLength is the shortest password accepted.
const MinPasswordLength = 6

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

// FindByLogin finds the account for what was typed on the login screen: a
// phone number (any common format) or a username. A phone-looking value that
// matches no phone is still tried as a username.
func (r *Repository) FindByLogin(login string) (*User, error) {
	login = strings.TrimSpace(login)
	if phone, ok := NormalizePhone(login); ok {
		var user User
		if err := r.db.Where("phone = ?", phone).Limit(1).Find(&user).Error; err != nil {
			return nil, err
		}
		if user.ID != uuid.Nil {
			return &user, nil
		}
	}
	return r.FindByUsername(login)
}

// phoneUsedByOther reports whether another account than userID has phone.
func (r *Repository) phoneUsedByOther(phone, userID string) (bool, error) {
	var count int64
	q := r.db.Model(&User{}).Where("phone = ?", phone)
	if userID != "" {
		q = q.Where("id <> ?", userID)
	}
	err := q.Count(&count).Error
	return count > 0, err
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
	user, err := s.repo.FindByLogin(req.Username)
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
	if req.Phone != nil {
		if user.Phone, err = s.checkPhone(*req.Phone, ""); err != nil {
			return nil, err
		}
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
	if req.Phone != nil {
		if user.Phone, err = s.checkPhone(*req.Phone, id); err != nil {
			return nil, err
		}
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

// ChangePassword sets a new password for userID after checking the current
// one. The new password must have MinPasswordLength characters and differ
// from the current one.
func (s *Service) ChangePassword(userID, current, next string) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(current)) != nil {
		return ErrWrongPassword
	}
	if len([]rune(next)) < MinPasswordLength || next == current {
		return ErrWeakPassword
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hash)
	return s.repo.Update(user)
}

// EnsureDefaultOwner creates the owner account when missing. phone (may be
// empty) is set on a new owner, and on an existing owner without a phone.
func (s *Service) EnsureDefaultOwner(username, password, fullName, phone string) error {
	existing, err := s.repo.FindByUsername(username)
	if err == nil {
		if phone != "" && existing.Phone == nil {
			_, err = s.UpdateUser(existing.ID.String(), UpdateUserRequest{Phone: &phone})
		}
		return err
	}
	if !errors.Is(err, ErrUserNotFound) {
		return err
	}
	req := CreateUserRequest{
		Username: username,
		Password: password,
		FullName: fullName,
		Role:     RoleOwner,
	}
	if phone != "" {
		req.Phone = &phone
	}
	_, err = s.CreateUser(req)
	return err
}

// UpdateProfile lets a signed-in user change their own name and phone (not
// role or active state). nil leaves a field unchanged; phone "" removes it.
func (s *Service) UpdateProfile(userID string, fullName, phone *string) (*User, error) {
	if fullName != nil {
		trimmed := strings.TrimSpace(*fullName)
		fullName = &trimmed
	}
	return s.UpdateUser(userID, UpdateUserRequest{FullName: fullName, Phone: phone})
}

// checkPhone validates and normalises a phone for userID ("" for a new user):
// "" → nil (no phone), otherwise ErrInvalidPhone / ErrPhoneTaken.
func (s *Service) checkPhone(raw, userID string) (*string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	phone, ok := NormalizePhone(raw)
	if !ok {
		return nil, ErrInvalidPhone
	}
	taken, err := s.repo.phoneUsedByOther(phone, userID)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, ErrPhoneTaken
	}
	return &phone, nil
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
		Phone:    u.Phone,
		FullName: u.FullName,
		Role:     u.Role,
	}
}
