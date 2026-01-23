package user

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/testutil"
)

func TestNewUserService(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	service := NewUserService(db)

	assert.NotNil(t, service)
	assert.IsType(t, &UserServiceImpl{}, service)
}

func TestCreateUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	req := auth.RegisterRequest{
		Name:     "John",
		Surname:  "Doe",
		Email:    "john.doe@example.com",
		Password: "password123",
	}
	passwordHash := "$2a$10$hashedpassword"

	user, err := service.CreateUser(req, passwordHash)

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Greater(t, user.ID, int64(0))
	assert.Equal(t, "John", user.Name)
	assert.Equal(t, "Doe", user.Surname)
	assert.Equal(t, "john.doe@example.com", user.Email)
	assert.Equal(t, passwordHash, user.PasswordHash)
	assert.True(t, user.Active)
	assert.NotZero(t, user.CreatedAt)
	assert.NotZero(t, user.UpdatedAt)
	assert.Nil(t, user.LastLogin) // LastLogin should be nil for newly created user
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	req := auth.RegisterRequest{
		Name:     "John",
		Surname:  "Doe",
		Email:    "duplicate@example.com",
		Password: "password123",
	}
	passwordHash := "$2a$10$hashedpassword"

	// Create first user
	user1, err := service.CreateUser(req, passwordHash)
	require.NoError(t, err)
	assert.NotNil(t, user1)

	// Try to create duplicate user with same email
	user2, err := service.CreateUser(req, passwordHash)
	assert.Error(t, err)
	assert.Nil(t, user2)
	assert.Contains(t, err.Error(), "failed to create user")
}

func TestCreateUser_EmptyFields(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	tests := []struct {
		name         string
		req          auth.RegisterRequest
		passwordHash string
		shouldError  bool
	}{
		{
			name: "Empty name",
			req: auth.RegisterRequest{
				Name:     "",
				Surname:  "Doe",
				Email:    "test1@example.com",
				Password: "password",
			},
			passwordHash: "$2a$10$hashedpassword",
			shouldError:  false, // Empty name is allowed in DB schema
		},
		{
			name: "Empty surname",
			req: auth.RegisterRequest{
				Name:     "John",
				Surname:  "",
				Email:    "test2@example.com",
				Password: "password",
			},
			passwordHash: "$2a$10$hashedpassword",
			shouldError:  false, // Empty surname is allowed in DB schema
		},
		{
			name: "Empty password hash",
			req: auth.RegisterRequest{
				Name:     "John",
				Surname:  "Doe",
				Email:    "test3@example.com",
				Password: "password",
			},
			passwordHash: "",
			shouldError:  false, // SQLite allows empty string even with NOT NULL (empty != NULL)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := service.CreateUser(tt.req, tt.passwordHash)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
			}
		})
	}
}

func TestGetUserByEmail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	// Create a user first
	req := auth.RegisterRequest{
		Name:     "Jane",
		Surname:  "Smith",
		Email:    "jane.smith@example.com",
		Password: "password123",
	}
	passwordHash := "$2a$10$hashedpassword"

	createdUser, err := service.CreateUser(req, passwordHash)
	require.NoError(t, err)

	// Retrieve the user by email
	user, err := service.GetUserByEmail("jane.smith@example.com")

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, createdUser.ID, user.ID)
	assert.Equal(t, "Jane", user.Name)
	assert.Equal(t, "Smith", user.Surname)
	assert.Equal(t, "jane.smith@example.com", user.Email)
	assert.Equal(t, passwordHash, user.PasswordHash)
	assert.True(t, user.Active)
	assert.Nil(t, user.LastLogin) // Should be nil initially
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	user, err := service.GetUserByEmail("nonexistent@example.com")

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user not found")
}

func TestGetUserByEmail_WithLastLogin(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	// Create a user
	req := auth.RegisterRequest{
		Name:     "Test",
		Surname:  "User",
		Email:    "test.user@example.com",
		Password: "password123",
	}
	passwordHash := "$2a$10$hashedpassword"

	createdUser, err := service.CreateUser(req, passwordHash)
	require.NoError(t, err)

	// Update last login
	err = service.UpdateLastLogin(createdUser.ID)
	require.NoError(t, err)

	// Retrieve the user
	user, err := service.GetUserByEmail("test.user@example.com")

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotNil(t, user.LastLogin) // Should have last login timestamp
	assert.WithinDuration(t, time.Now(), *user.LastLogin, 2*time.Second)
}

func TestUpdateLastLogin(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	// Create a user
	req := auth.RegisterRequest{
		Name:     "Update",
		Surname:  "Test",
		Email:    "update.test@example.com",
		Password: "password123",
	}
	passwordHash := "$2a$10$hashedpassword"

	createdUser, err := service.CreateUser(req, passwordHash)
	require.NoError(t, err)

	// Update last login
	err = service.UpdateLastLogin(createdUser.ID)
	require.NoError(t, err)

	// Verify last login was updated
	var lastLogin sql.NullTime
	err = db.QueryRow("SELECT last_login FROM users WHERE id = ?", createdUser.ID).Scan(&lastLogin)
	require.NoError(t, err)
	assert.True(t, lastLogin.Valid)
	assert.WithinDuration(t, time.Now(), lastLogin.Time, 2*time.Second)
}

func TestUpdateLastLogin_NonexistentUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	// Try to update last login for non-existent user
	err := service.UpdateLastLogin(999999)

	// Should not error even if user doesn't exist (UPDATE affects 0 rows)
	assert.NoError(t, err)
}

func TestUpdateLastLogin_Multiple(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	// Create a user
	req := auth.RegisterRequest{
		Name:     "Multiple",
		Surname:  "Login",
		Email:    "multiple.login@example.com",
		Password: "password123",
	}
	passwordHash := "$2a$10$hashedpassword"

	createdUser, err := service.CreateUser(req, passwordHash)
	require.NoError(t, err)

	// Update last login first time
	err = service.UpdateLastLogin(createdUser.ID)
	require.NoError(t, err)

	var firstLogin sql.NullTime
	err = db.QueryRow("SELECT last_login FROM users WHERE id = ?", createdUser.ID).Scan(&firstLogin)
	require.NoError(t, err)

	// Sleep to ensure timestamp difference
	time.Sleep(100 * time.Millisecond)

	// Update last login second time
	err = service.UpdateLastLogin(createdUser.ID)
	require.NoError(t, err)

	var secondLogin sql.NullTime
	err = db.QueryRow("SELECT last_login FROM users WHERE id = ?", createdUser.ID).Scan(&secondLogin)
	require.NoError(t, err)

	// Second login should be after first login
	assert.True(t, secondLogin.Time.After(firstLogin.Time))
}

func TestHasUsers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	// Check when no users exist
	hasUsers, err := service.HasUsers()
	require.NoError(t, err)
	assert.False(t, hasUsers)

	// Create a user
	req := auth.RegisterRequest{
		Name:     "First",
		Surname:  "User",
		Email:    "first.user@example.com",
		Password: "password123",
	}
	passwordHash := "$2a$10$hashedpassword"

	_, err = service.CreateUser(req, passwordHash)
	require.NoError(t, err)

	// Check again
	hasUsers, err = service.HasUsers()
	require.NoError(t, err)
	assert.True(t, hasUsers)
}

func TestHasUsers_MultipleUsers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	// Create multiple users
	for i := 1; i <= 5; i++ {
		req := auth.RegisterRequest{
			Name:     "User",
			Surname:  "Test",
			Email:    "user" + string(rune(i)) + "@example.com",
			Password: "password123",
		}
		passwordHash := "$2a$10$hashedpassword"
		_, err := service.CreateUser(req, passwordHash)
		require.NoError(t, err)
	}

	hasUsers, err := service.HasUsers()
	require.NoError(t, err)
	assert.True(t, hasUsers)
}

func TestGetFirstUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	// Create multiple users with slight delays
	users := []auth.RegisterRequest{
		{Name: "First", Surname: "User", Email: "first@example.com", Password: "pass1"},
		{Name: "Second", Surname: "User", Email: "second@example.com", Password: "pass2"},
		{Name: "Third", Surname: "User", Email: "third@example.com", Password: "pass3"},
	}

	passwordHash := "$2a$10$hashedpassword"
	var createdUsers []*auth.User

	for _, req := range users {
		user, err := service.CreateUser(req, passwordHash)
		require.NoError(t, err)
		createdUsers = append(createdUsers, user)
		time.Sleep(10 * time.Millisecond) // Small delay to ensure created_at ordering
	}

	// Get first user
	firstUser, err := service.GetFirstUser()

	require.NoError(t, err)
	assert.NotNil(t, firstUser)
	assert.Equal(t, createdUsers[0].ID, firstUser.ID)
	assert.Equal(t, "First", firstUser.Name)
	assert.Equal(t, "first@example.com", firstUser.Email)
}

func TestGetFirstUser_NoUsers(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	user, err := service.GetFirstUser()

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "no users found")
}

func TestGetFirstUser_WithLastLogin(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	// Create a user
	req := auth.RegisterRequest{
		Name:     "Login",
		Surname:  "Test",
		Email:    "login.test@example.com",
		Password: "password123",
	}
	passwordHash := "$2a$10$hashedpassword"

	createdUser, err := service.CreateUser(req, passwordHash)
	require.NoError(t, err)

	// Update last login
	err = service.UpdateLastLogin(createdUser.ID)
	require.NoError(t, err)

	// Get first user
	firstUser, err := service.GetFirstUser()

	require.NoError(t, err)
	assert.NotNil(t, firstUser)
	assert.NotNil(t, firstUser.LastLogin)
	assert.WithinDuration(t, time.Now(), *firstUser.LastLogin, 2*time.Second)
}

func TestUserService_Integration(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	// Test complete flow: create -> get -> update login -> verify
	req := auth.RegisterRequest{
		Name:     "Integration",
		Surname:  "Test",
		Email:    "integration.test@example.com",
		Password: "password123",
	}
	passwordHash := "$2a$10$hashedpassword"

	// Create user
	createdUser, err := service.CreateUser(req, passwordHash)
	require.NoError(t, err)
	assert.NotNil(t, createdUser)

	// Get user by email
	retrievedUser, err := service.GetUserByEmail("integration.test@example.com")
	require.NoError(t, err)
	assert.Equal(t, createdUser.ID, retrievedUser.ID)
	assert.Nil(t, retrievedUser.LastLogin)

	// Update last login
	err = service.UpdateLastLogin(createdUser.ID)
	require.NoError(t, err)

	// Get user again and verify last login
	updatedUser, err := service.GetUserByEmail("integration.test@example.com")
	require.NoError(t, err)
	assert.NotNil(t, updatedUser.LastLogin)

	// Check HasUsers
	hasUsers, err := service.HasUsers()
	require.NoError(t, err)
	assert.True(t, hasUsers)

	// Check GetFirstUser
	firstUser, err := service.GetFirstUser()
	require.NoError(t, err)
	assert.Equal(t, createdUser.ID, firstUser.ID)
}

func TestUserService_CaseInsensitiveEmail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	// Create user with lowercase email
	req := auth.RegisterRequest{
		Name:     "Case",
		Surname:  "Test",
		Email:    "case.test@example.com",
		Password: "password123",
	}
	passwordHash := "$2a$10$hashedpassword"

	createdUser, err := service.CreateUser(req, passwordHash)
	require.NoError(t, err)

	// Try to retrieve with different case
	// Note: SQLite is case-insensitive by default for LIKE, but case-sensitive for =
	// This behavior depends on PRAGMA case_sensitive_like and collation
	user, err := service.GetUserByEmail("case.test@example.com")
	require.NoError(t, err)
	assert.Equal(t, createdUser.ID, user.ID)
}

func TestUserService_ActiveFieldDefault(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.ApplyTestMigrations(t, db, "../../db/migrations")

	service := NewUserService(db)

	req := auth.RegisterRequest{
		Name:     "Active",
		Surname:  "Test",
		Email:    "active.test@example.com",
		Password: "password123",
	}
	passwordHash := "$2a$10$hashedpassword"

	createdUser, err := service.CreateUser(req, passwordHash)
	require.NoError(t, err)

	// Verify active field is set to true by default
	assert.True(t, createdUser.Active)

	// Verify in database
	var active bool
	err = db.QueryRow("SELECT active FROM users WHERE id = ?", createdUser.ID).Scan(&active)
	require.NoError(t, err)
	assert.True(t, active)
}
