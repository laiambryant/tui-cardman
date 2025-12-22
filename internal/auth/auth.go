package auth

import (
	"fmt"
	"regexp"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// ValidateEmail checks if email format is valid
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

// ValidatePassword checks password meets requirements
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	hasUpper := false
	hasSpecial := false
	for _, char := range password {
		if unicode.IsUpper(char) {
			hasUpper = true
		}
		if unicode.IsPunct(char) || unicode.IsSymbol(char) {
			hasSpecial = true
		}
	}
	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// HashPassword generates bcrypt hash from password
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword checks if password matches hash
func VerifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// Service provides authentication operations
type Service struct {
	db interface {
		CreateUser(req RegisterRequest, passwordHash string) (*User, error)
		GetUserByEmail(email string) (*User, error)
		UpdateLastLogin(userID int64) error
	}
}

// NewService creates a new auth service
func NewService(db interface {
	CreateUser(req RegisterRequest, passwordHash string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	UpdateLastLogin(userID int64) error
}) *Service {
	return &Service{db: db}
}

// Register creates a new user account
func (s *Service) Register(req RegisterRequest) (*User, error) {
	// Validate email
	if err := ValidateEmail(req.Email); err != nil {
		return nil, err
	}

	// Validate password
	if err := ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// Check if user already exists
	_, err := s.db.GetUserByEmail(req.Email)
	if err == nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	// Hash password
	passwordHash, err := HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Create user
	user, err := s.db.CreateUser(req, passwordHash)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// Login authenticates a user
func (s *Service) Login(req LoginRequest) (*User, error) {
	// Get user by email
	user, err := s.db.GetUserByEmail(req.Email)
	if err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Check if user is active
	if !user.Active {
		return nil, fmt.Errorf("user account is inactive")
	}

	// Verify password
	if err := VerifyPassword(req.Password, user.PasswordHash); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	// Update last login
	if err := s.db.UpdateLastLogin(user.ID); err != nil {
		// Log error but don't fail login
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	return user, nil
}
