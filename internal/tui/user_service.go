package tui

import (
	"database/sql"
	"fmt"
	"time"

	"gihtub.com/laiambryant/tui-cardman/internal/auth"
)

// IUserService defines the interface for user-related operations
type IUserService interface {
	CreateUser(req auth.RegisterRequest, passwordHash string) (*auth.User, error)
	GetUserByEmail(email string) (*auth.User, error)
	UpdateLastLogin(userID int64) error
}

// UserServiceImpl implements the IUserService interface
type UserServiceImpl struct {
	db *sql.DB
}

// NewUserService creates a new instance of UserServiceImpl
func NewUserService(db *sql.DB) IUserService {
	return &UserServiceImpl{db: db}
}

const (
	insertUserQuery = `
		INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active)
		VALUES (?, ?, ?, ?, ?, ?, 1)
	`
	selectUserByEmailQuery = `
		SELECT id, name, surname, email, password_hash, created_at, updated_at, last_login, active
		FROM users
		WHERE email = ?
	`
	updateLastLoginQuery = `UPDATE users SET last_login = ? WHERE id = ?`
)

// CreateUser inserts a new user into the database
func (s *UserServiceImpl) CreateUser(req auth.RegisterRequest, passwordHash string) (*auth.User, error) {
	now := time.Now()
	result, err := s.db.Exec(insertUserQuery, req.Name, req.Surname, req.Email, passwordHash, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user id: %w", err)
	}

	return &auth.User{
		ID:           id,
		Name:         req.Name,
		Surname:      req.Surname,
		Email:        req.Email,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
		Active:       true,
	}, nil
}

// GetUserByEmail retrieves a user by email
func (s *UserServiceImpl) GetUserByEmail(email string) (*auth.User, error) {
	var user auth.User
	var lastLogin sql.NullTime

	err := s.db.QueryRow(selectUserByEmailQuery, email).Scan(
		&user.ID,
		&user.Name,
		&user.Surname,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
		&lastLogin,
		&user.Active,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	return &user, nil
}

// UpdateLastLogin updates the last_login timestamp for a user
func (s *UserServiceImpl) UpdateLastLogin(userID int64) error {
	_, err := s.db.Exec(updateLastLoginQuery, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}
