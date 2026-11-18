package auth

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string       `json:"token"`
	ExpiresIn int64        `json:"expires_in"`
	User      UserResponse `json:"user"`
}

type UserResponse struct {
	ID       string  `json:"id"`
	Username string  `json:"username"`
	Phone    *string `json:"phone"`
	FullName string  `json:"full_name"`
	Role     Role    `json:"role"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	// Optional; any common format, stored normalised.
	Phone    *string `json:"phone"`
	Password string  `json:"password"`
	FullName string  `json:"full_name"`
	Role     Role    `json:"role"`
}

type UpdateUserRequest struct {
	FullName *string `json:"full_name"`
	// "" removes the phone.
	Phone    *string `json:"phone"`
	Password *string `json:"password"`
	Role     *Role   `json:"role"`
	Active   *bool   `json:"active"`
}
