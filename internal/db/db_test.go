package db

import (
	"database/sql"
	"testing"
	"time"

	"github.com/laiambryant/gotestutils/ctesting"
	"github.com/laiambryant/tui-cardman/internal/auth"
	"github.com/laiambryant/tui-cardman/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenDB tests database connection opening
func TestOpenDB(t *testing.T) {
	tests := []struct {
		name    string
		dsn     string
		wantErr bool
	}{
		{
			name:    "in-memory database",
			dsn:     ":memory:",
			wantErr: false,
		},
		{
			name:    "file-based database",
			dsn:     "file::memory:?cache=shared",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := OpenDB(tt.dsn)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, db)

			// Test ping
			err = db.Ping()
			require.NoError(t, err)

			// Clean up
			err = db.Close()
			require.NoError(t, err)
		})
	}
}

// TestCreateUser tests user creation in database
func TestCreateUser(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	testutil.CreateTestSchema(t, db)

	tests := []struct {
		name          string
		req           auth.RegisterRequest
		passwordHash  string
		setup         func()
		expectedError string
	}{
		{
			name: "successful user creation",
			req: auth.RegisterRequest{
				Name:     "John",
				Surname:  "Doe",
				Email:    "john.doe@example.com",
				Password: "ignored",
			},
			passwordHash:  "$2a$10$validhashedpassword",
			setup:         func() {},
			expectedError: "",
		},
		{
			name: "duplicate email",
			req: auth.RegisterRequest{
				Name:     "Jane",
				Surname:  "Smith",
				Email:    "john.doe@example.com",
				Password: "ignored",
			},
			passwordHash: "$2a$10$anotherhashedpassword",
			setup: func() {
				_, err := db.Exec(
					"INSERT INTO users (name, surname, email, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
					"Existing", "User", "john.doe@example.com", "$2a$10$hash", time.Now(), time.Now(),
				)
				require.NoError(t, err)
			},
			expectedError: "UNIQUE constraint failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			user, err := CreateUser(db, tt.req, tt.passwordHash)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.NotZero(t, user.ID)
				assert.Equal(t, tt.req.Name, user.Name)
				assert.Equal(t, tt.req.Surname, user.Surname)
				assert.Equal(t, tt.req.Email, user.Email)
				assert.Equal(t, tt.passwordHash, user.PasswordHash)
				assert.True(t, user.Active)
				assert.False(t, user.CreatedAt.IsZero())
				assert.False(t, user.UpdatedAt.IsZero())
			}

			// Clean up for next iteration
			testutil.TruncateAllTables(t, db)
		})
	}
}

// TestCreateUser_Characterization tests user creation with characterization testing
func TestCreateUser_Characterization(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	testutil.CreateTestSchema(t, db)

	req := auth.RegisterRequest{
		Name:     "Alice",
		Surname:  "Johnson",
		Email:    "alice@example.com",
		Password: "ignored",
	}
	passwordHash := "$2a$10$testhashedpassword"

	tests := []ctesting.CharacterizationTest[int64]{
		// Successful user creation - verify ID is returned
		ctesting.NewCharacterizationTest(
			int64(1), // First user gets ID 1
			nil,
			func() (int64, error) {
				user, err := CreateUser(db, req, passwordHash)
				if err != nil {
					return 0, err
				}
				// Verify user fields manually
				assert.Equal(t, "Alice", user.Name)
				assert.Equal(t, "Johnson", user.Surname)
				assert.Equal(t, "alice@example.com", user.Email)
				assert.Equal(t, passwordHash, user.PasswordHash)
				assert.True(t, user.Active)
				return user.ID, nil
			},
		),
	}

	// Run characterization tests
	ctesting.VerifyCharacterizationTestsAndResults(t, tests, false)
}

// TestGetUserByEmail tests user retrieval by email
func TestGetUserByEmail(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	testutil.CreateTestSchema(t, db)

	// Insert test user
	now := time.Now()
	_, err := db.Exec(
		"INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"Test", "User", "test@example.com", "$2a$10$hash", now, now, 1,
	)
	require.NoError(t, err)

	tests := []struct {
		name          string
		email         string
		expectedError string
	}{
		{
			name:          "existing user",
			email:         "test@example.com",
			expectedError: "",
		},
		{
			name:          "non-existent user",
			email:         "nonexistent@example.com",
			expectedError: "user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := GetUserByEmail(db, tt.email)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tt.email, user.Email)
				assert.Equal(t, "Test", user.Name)
				assert.Equal(t, "User", user.Surname)
				assert.True(t, user.Active)
			}
		})
	}
}

// TestGetUserByEmail_Characterization tests email lookup with characterization testing
func TestGetUserByEmail_Characterization(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	testutil.CreateTestSchema(t, db)

	// Insert test user
	now := time.Now()
	_, err := db.Exec(
		"INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"Bob", "Smith", "bob@example.com", "$2a$10$bobhash", now, now, 1,
	)
	require.NoError(t, err)

	tests := []ctesting.CharacterizationTest[string]{
		// Successful lookup
		ctesting.NewCharacterizationTest(
			"bob@example.com", // Expected email
			nil,
			func() (string, error) {
				user, err := GetUserByEmail(db, "bob@example.com")
				if err != nil {
					return "", err
				}
				return user.Email, nil
			},
		),
		// User not found
		ctesting.NewCharacterizationTest(
			"",
			nil,
			func() (string, error) {
				user, err := GetUserByEmail(db, "nonexistent@example.com")
				if err != nil {
					return "", nil // Expected error
				}
				return user.Email, nil
			},
		),
	}

	ctesting.VerifyCharacterizationTestsAndResults(t, tests, false)
}

// TestUpdateLastLogin tests updating last login timestamp
func TestUpdateLastLogin(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	testutil.CreateTestSchema(t, db)

	// Insert test user
	now := time.Now()
	result, err := db.Exec(
		"INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"Test", "User", "test@example.com", "$2a$10$hash", now, now, 1,
	)
	require.NoError(t, err)

	userID, err := result.LastInsertId()
	require.NoError(t, err)

	// Update last login
	err = UpdateLastLogin(db, userID)
	require.NoError(t, err)

	// Verify last_login was set
	var lastLogin sql.NullTime
	err = db.QueryRow("SELECT last_login FROM users WHERE id = ?", userID).Scan(&lastLogin)
	require.NoError(t, err)
	assert.True(t, lastLogin.Valid)
	assert.False(t, lastLogin.Time.IsZero())

	// Test with non-existent user (should not error but also not update anything)
	err = UpdateLastLogin(db, 99999)
	require.NoError(t, err)
}

// TestUpdateLastLogin_Characterization tests last login update with characterization testing
func TestUpdateLastLogin_Characterization(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)
	testutil.CreateTestSchema(t, db)

	// Insert test user
	now := time.Now()
	result, err := db.Exec(
		"INSERT INTO users (name, surname, email, password_hash, created_at, updated_at, active) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"Charlie", "Brown", "charlie@example.com", "$2a$10$charliehash", now, now, 1,
	)
	require.NoError(t, err)

	userID, err := result.LastInsertId()
	require.NoError(t, err)

	tests := []ctesting.CharacterizationTest[error]{
		// Successful update
		ctesting.NewCharacterizationTest(
			nil, // Expected error (none)
			nil,
			func() (error, error) {
				return UpdateLastLogin(db, userID), nil
			},
		),
		// Update with non-existent user (no error expected)
		ctesting.NewCharacterizationTest(
			nil,
			nil,
			func() (error, error) {
				return UpdateLastLogin(db, 99999), nil
			},
		),
	}

	ctesting.VerifyCharacterizationTestsAndResults(t, tests, false)
}

// TestDatabaseConnection_Characterization tests database connection behavior
func TestDatabaseConnection_Characterization(t *testing.T) {
	tests := []ctesting.CharacterizationTest[error]{
		// Successful connection
		ctesting.NewCharacterizationTest(
			nil, // Expected error (none)
			nil,
			func() (error, error) {
				db, err := OpenDB(":memory:")
				if err != nil {
					return err, nil
				}
				defer db.Close()
				return db.Ping(), nil
			},
		),
	}

	ctesting.VerifyCharacterizationTestsAndResults(t, tests, false)
}
