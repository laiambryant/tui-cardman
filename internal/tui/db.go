package tui

import (
	"database/sql"
	"fmt"
	"time"

	"gihtub.com/laiambryant/tui-cardman/internal/auth"
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

func createUser(db *sql.DB, req auth.RegisterRequest, passwordHash string) (*auth.User, error) {
	query := insertUserQuery
	now := time.Now()
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

// getUserByEmail retrieves a user by email
func getUserByEmail(db *sql.DB, email string) (*auth.User, error) {
	query := selectUserByEmailQuery
	var user auth.User
	var lastLogin sql.NullTime
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

// updateLastLogin updates the last_login timestamp for a user
func updateLastLogin(db *sql.DB, userID int64) error {
	_, err := db.Exec(updateLastLoginQuery, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}
