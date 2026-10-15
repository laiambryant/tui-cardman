// Package user provides services for managing user accounts.
package user

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/db"
)

// UserService defines the interface for user-related operations
type UserService interface {
	CreateUser(req auth.RegisterRequest, passwordHash string) (*auth.User, error)
	GetUserByEmail(email string) (*auth.User, error)
	UpdateLastLogin(userID int64) error
	HasUsers() (bool, error)
	GetFirstUser() (*auth.User, error)
}

// UserServiceImpl implements the UserService interface
type UserServiceImpl struct {
	db *sql.DB
}

// NewUserService creates a new instance of UserServiceImpl
func NewUserService(db *sql.DB) UserService {
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
	selectFirstUserQuery = `
		SELECT id, name, surname, email, password_hash, created_at, updated_at, last_login, active
		FROM users
		ORDER BY created_at ASC
		LIMIT 1
	`
	updateLastLoginQuery = `UPDATE users SET last_login = ? WHERE id = ?`
)

// CreateUser inserts a new user into the database
func (s *UserServiceImpl) CreateUser(req auth.RegisterRequest, passwordHash string) (*auth.User, error) {
	now := time.Now()
	result, err := db.Exec(s.db, insertUserQuery, req.Name, req.Surname, req.Email, passwordHash, now, now)
	if err != nil {
		slog.Error("failed to create user", "email", req.Email, "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		slog.Error("failed to get last insert id for user", "email", req.Email, "error", err)
		return nil, fmt.Errorf("failed to get user id: %w", err)
	}

	user := &auth.User{
		ID:           id,
		Name:         req.Name,
		Surname:      req.Surname,
		Email:        req.Email,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
		Active:       true,
	}
	slog.Debug("created user", "user_id", id, "email", req.Email)
	return user, nil
}

// GetUserByEmail retrieves a user by email
func (s *UserServiceImpl) GetUserByEmail(email string) (*auth.User, error) {
	user, err := scanUser(db.QueryRow(s.db, selectUserByEmailQuery, email))
	if err == sql.ErrNoRows {
		slog.Debug("user not found by email", "email", email)
		return nil, ErrUserNotFound
	}
	if err != nil {
		slog.Error("failed to get user by email", "email", email, "error", err)
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

// UpdateLastLogin updates the last_login timestamp for a user
func (s *UserServiceImpl) UpdateLastLogin(userID int64) error {
	_, err := db.Exec(s.db, updateLastLoginQuery, time.Now(), userID)
	if err != nil {
		slog.Error("failed to update last login", "user_id", userID, "error", err)
		return fmt.Errorf("failed to update last login: %w", err)
	}
	slog.Debug("updated last login", "user_id", userID)
	return nil
}

// HasUsers checks if any users exist in the database
func (s *UserServiceImpl) HasUsers() (bool, error) {
	var count int
	if err := db.QueryRow(s.db, "SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return false, fmt.Errorf("failed to count users: %w", err)
	}
	return count > 0, nil
}

// GetFirstUser retrieves the first user (by creation date) from the database
func (s *UserServiceImpl) GetFirstUser() (*auth.User, error) {
	user, err := scanUser(db.QueryRow(s.db, selectFirstUserQuery))
	if err == sql.ErrNoRows {
		return nil, ErrNoUsersFound
	}
	if err != nil {
		slog.Error("failed to get first user", "error", err)
		return nil, fmt.Errorf("failed to get first user: %w", err)
	}
	return user, nil
}

func scanUser(row *sql.Row) (*auth.User, error) {
	var user auth.User
	var lastLogin sql.NullTime
	err := row.Scan(&user.ID, &user.Name, &user.Surname, &user.Email, &user.PasswordHash,
		&user.CreatedAt, &user.UpdatedAt, &lastLogin, &user.Active)
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}
	return &user, nil
}
