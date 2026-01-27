package user

import (
	"database/sql"
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
		return nil, &FailedToCreateUserError{Err: err}
	}

	id, err := result.LastInsertId()
	if err != nil {
		slog.Error("failed to get last insert id for user", "email", req.Email, "error", err)
		return nil, &FailedToGetUserIDError{Err: err}
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
	var user auth.User
	var lastLogin sql.NullTime

	err := db.QueryRow(s.db, selectUserByEmailQuery, email).Scan(
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
			slog.Debug("user not found by email", "email", email)
			return nil, &UserNotFoundError{}
		}
		slog.Error("failed to get user by email", "email", email, "error", err)
		return nil, &FailedToGetUserError{Err: err}
	}

	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	slog.Debug("found user by email", "email", email, "user_id", user.ID)
	return &user, nil
}

// UpdateLastLogin updates the last_login timestamp for a user
func (s *UserServiceImpl) UpdateLastLogin(userID int64) error {
	_, err := db.Exec(s.db, updateLastLoginQuery, time.Now(), userID)
	if err != nil {
		slog.Error("failed to update last login", "user_id", userID, "error", err)
		return &FailedToUpdateLastLoginError{Err: err}
	}
	slog.Debug("updated last login", "user_id", userID)
	return nil
}

// HasUsers checks if any users exist in the database
func (s *UserServiceImpl) HasUsers() (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM users"
	err := db.QueryRow(s.db, query).Scan(&count)
	if err != nil {
		slog.Error("failed to count users", "error", err)
		return false, &FailedToCountUsersError{Err: err}
	}
	hasUsers := count > 0
	slog.Debug("checked if users exist", "count", count, "has_users", hasUsers)
	return hasUsers, nil
}

// GetFirstUser retrieves the first user (by creation date) from the database
func (s *UserServiceImpl) GetFirstUser() (*auth.User, error) {
	var user auth.User
	var lastLogin sql.NullTime

	err := db.QueryRow(s.db, selectFirstUserQuery).Scan(
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
			slog.Debug("no users found")
			return nil, &NoUsersFoundError{}
		}
		slog.Error("failed to get first user", "error", err)
		return nil, &FailedToGetFirstUserError{Err: err}
	}

	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}

	slog.Debug("found first user", "user_id", user.ID, "email", user.Email)
	return &user, nil
}
