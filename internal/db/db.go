package db

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/logging"
	_ "github.com/mattn/go-sqlite3"
)

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

func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, db.Ping()
}

// CreateUser inserts a new user into the database
func CreateUser(db *sql.DB, req auth.RegisterRequest, passwordHash string) (*auth.User, error) {
	query := insertUserQuery
	now := time.Now()
	slog.Debug("exec query", "query", logging.SanitizeQuery(query), "args", []any{req.Name, req.Surname, req.Email, passwordHash, now, now})
	result, err := db.Exec(query, req.Name, req.Surname, req.Email, passwordHash, now, now)
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
func GetUserByEmail(db *sql.DB, email string) (*auth.User, error) {
	query := selectUserByEmailQuery
	var user auth.User
	var lastLogin sql.NullTime

	slog.Debug("query row", "query", logging.SanitizeQuery(query), "args", []any{email})
	err := db.QueryRow(query, email).Scan(
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
func UpdateLastLogin(db *sql.DB, userID int64) error {
	args := []any{time.Now(), userID}
	slog.Debug("exec query", "query", logging.SanitizeQuery(updateLastLoginQuery), "args", args)
	_, err := db.Exec(updateLastLoginQuery, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}
