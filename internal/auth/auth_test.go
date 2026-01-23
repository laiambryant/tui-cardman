package auth

import (
	"errors"
	"testing"

	"github.com/laiambryant/gotestutils/ctesting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateEmail_Characterization tests email validation with characterization testing
func TestValidateEmail_Characterization(t *testing.T) {
	tests := []ctesting.CharacterizationTest[error]{
		// Valid email addresses
		ctesting.NewCharacterizationTest(
			nil, // Expected output (no error)
			nil, // Expected error
			func() (error, error) {
				return ValidateEmail("test@example.com"), nil
			},
		),
		ctesting.NewCharacterizationTest(
			nil,
			nil,
			func() (error, error) {
				return ValidateEmail("user.name+tag@example.co.uk"), nil
			},
		),
		ctesting.NewCharacterizationTest(
			nil,
			nil,
			func() (error, error) {
				return ValidateEmail("test_email@test-domain.com"), nil
			},
		),
		// Empty email
		ctesting.NewCharacterizationTest(
			errors.New("email is required"),
			nil,
			func() (error, error) {
				return ValidateEmail(""), nil
			},
		),
		// Invalid email formats
		ctesting.NewCharacterizationTest(
			errors.New("invalid email format"),
			nil,
			func() (error, error) {
				return ValidateEmail("invalid-email"), nil
			},
		),
		ctesting.NewCharacterizationTest(
			errors.New("invalid email format"),
			nil,
			func() (error, error) {
				return ValidateEmail("@example.com"), nil
			},
		),
		ctesting.NewCharacterizationTest(
			errors.New("invalid email format"),
			nil,
			func() (error, error) {
				return ValidateEmail("test@"), nil
			},
		),
		ctesting.NewCharacterizationTest(
			errors.New("invalid email format"),
			nil,
			func() (error, error) {
				return ValidateEmail("test@.com"), nil
			},
		),
		ctesting.NewCharacterizationTest(
			errors.New("invalid email format"),
			nil,
			func() (error, error) {
				return ValidateEmail("test @example.com"), nil
			},
		),
	}

	ctesting.VerifyCharacterizationTestsAndResults(t, tests, false)
}

// TestValidatePassword_Characterization tests password validation with characterization testing
func TestValidatePassword_Characterization(t *testing.T) {
	tests := []ctesting.CharacterizationTest[error]{
		// Valid passwords
		ctesting.NewCharacterizationTest(
			nil, // Expected output (no error)
			nil, // Expected error
			func() (error, error) {
				return ValidatePassword("SecureP@ss123"), nil
			},
		),
		ctesting.NewCharacterizationTest(
			nil,
			nil,
			func() (error, error) {
				return ValidatePassword("Minimum8!"), nil
			},
		),
		ctesting.NewCharacterizationTest(
			nil,
			nil,
			func() (error, error) {
				return ValidatePassword("Compl3x!P@ssw0rd"), nil
			},
		),
		// Too short
		ctesting.NewCharacterizationTest(
			errors.New("password must be at least 8 characters"),
			nil,
			func() (error, error) {
				return ValidatePassword("Short1!"), nil
			},
		),
		// Missing uppercase
		ctesting.NewCharacterizationTest(
			errors.New("password must contain at least one uppercase letter"),
			nil,
			func() (error, error) {
				return ValidatePassword("lowercase123!"), nil
			},
		),
		// Missing special character
		ctesting.NewCharacterizationTest(
			errors.New("password must contain at least one special character"),
			nil,
			func() (error, error) {
				return ValidatePassword("NoSpecial123"), nil
			},
		),
		// Empty password
		ctesting.NewCharacterizationTest(
			errors.New("password must be at least 8 characters"),
			nil,
			func() (error, error) {
				return ValidatePassword(""), nil
			},
		),
	}

	ctesting.VerifyCharacterizationTestsAndResults(t, tests, false)
}

// TestHashPassword tests password hashing functionality
func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "simple password",
			password: "TestPassword123!",
		},
		{
			name:     "complex password",
			password: "V3ry!C0mpl3x@P@ssw0rd#2025",
		},
		{
			name:     "minimum valid password",
			password: "Minimum8!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			require.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotEqual(t, tt.password, hash)

			// Verify the hash works with VerifyPassword
			err = VerifyPassword(tt.password, hash)
			assert.NoError(t, err)
		})
	}
}

// TestVerifyPassword tests password verification
func TestVerifyPassword(t *testing.T) {
	password := "TestPassword123!"
	hash, err := HashPassword(password)
	require.NoError(t, err)

	tests := []struct {
		name        string
		password    string
		hash        string
		shouldMatch bool
	}{
		{
			name:        "correct password",
			password:    password,
			hash:        hash,
			shouldMatch: true,
		},
		{
			name:        "wrong password",
			password:    "WrongPassword123!",
			hash:        hash,
			shouldMatch: false,
		},
		{
			name:        "empty password",
			password:    "",
			hash:        hash,
			shouldMatch: false,
		},
		{
			name:        "case sensitive mismatch",
			password:    "testpassword123!",
			hash:        hash,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyPassword(tt.password, tt.hash)
			if tt.shouldMatch {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

// Mock database for testing Service methods
type mockDB struct {
	users               map[string]*User
	createUserFunc      func(req RegisterRequest, passwordHash string) (*User, error)
	getUserByEmailFunc  func(email string) (*User, error)
	updateLastLoginFunc func(userID int64) error
}

func newMockDB() *mockDB {
	return &mockDB{
		users: make(map[string]*User),
	}
}

func (m *mockDB) CreateUser(req RegisterRequest, passwordHash string) (*User, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(req, passwordHash)
	}

	// Check if user exists
	if _, exists := m.users[req.Email]; exists {
		return nil, errors.New("user already exists")
	}

	user := &User{
		ID:           int64(len(m.users) + 1),
		Name:         req.Name,
		Surname:      req.Surname,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Active:       true,
	}
	m.users[req.Email] = user
	return user, nil
}

func (m *mockDB) GetUserByEmail(email string) (*User, error) {
	if m.getUserByEmailFunc != nil {
		return m.getUserByEmailFunc(email)
	}

	user, exists := m.users[email]
	if !exists {
		return nil, errors.New("user not found")
	}
	return user, nil
}

func (m *mockDB) UpdateLastLogin(userID int64) error {
	if m.updateLastLoginFunc != nil {
		return m.updateLastLoginFunc(userID)
	}
	return nil
}

// TestService_Register tests user registration
func TestService_Register(t *testing.T) {
	tests := []struct {
		name          string
		req           RegisterRequest
		mockSetup     func(*mockDB)
		expectedError string
	}{
		{
			name: "successful registration",
			req: RegisterRequest{
				Name:     "John",
				Surname:  "Doe",
				Email:    "john.doe@example.com",
				Password: "SecureP@ss123",
			},
			mockSetup:     func(m *mockDB) {},
			expectedError: "",
		},
		{
			name: "invalid email",
			req: RegisterRequest{
				Name:     "John",
				Surname:  "Doe",
				Email:    "invalid-email",
				Password: "SecureP@ss123",
			},
			mockSetup:     func(m *mockDB) {},
			expectedError: "invalid email format",
		},
		{
			name: "invalid password - too short",
			req: RegisterRequest{
				Name:     "John",
				Surname:  "Doe",
				Email:    "john.doe@example.com",
				Password: "Short1!",
			},
			mockSetup:     func(m *mockDB) {},
			expectedError: "password must be at least 8 characters",
		},
		{
			name: "invalid password - no uppercase",
			req: RegisterRequest{
				Name:     "John",
				Surname:  "Doe",
				Email:    "john.doe@example.com",
				Password: "nouppercas3!",
			},
			mockSetup:     func(m *mockDB) {},
			expectedError: "password must contain at least one uppercase letter",
		},
		{
			name: "invalid password - no special character",
			req: RegisterRequest{
				Name:     "John",
				Surname:  "Doe",
				Email:    "john.doe@example.com",
				Password: "NoSpecial123",
			},
			mockSetup:     func(m *mockDB) {},
			expectedError: "password must contain at least one special character",
		},
		{
			name: "duplicate user",
			req: RegisterRequest{
				Name:     "John",
				Surname:  "Doe",
				Email:    "existing@example.com",
				Password: "SecureP@ss123",
			},
			mockSetup: func(m *mockDB) {
				m.users["existing@example.com"] = &User{
					ID:     1,
					Email:  "existing@example.com",
					Active: true,
				}
			},
			expectedError: "user with this email already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := newMockDB()
			tt.mockSetup(mockDB)

			service := NewService(mockDB)
			user, err := service.Register(tt.req)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tt.req.Email, user.Email)
				assert.Equal(t, tt.req.Name, user.Name)
				assert.Equal(t, tt.req.Surname, user.Surname)
				assert.True(t, user.Active)
				assert.NotEmpty(t, user.PasswordHash)
				assert.NotEqual(t, tt.req.Password, user.PasswordHash)
			}
		})
	}
}

// TestService_Login tests user login
func TestService_Login(t *testing.T) {
	validPassword := "SecureP@ss123"
	validHash, _ := HashPassword(validPassword)

	tests := []struct {
		name          string
		req           LoginRequest
		mockSetup     func(*mockDB)
		expectedError string
	}{
		{
			name: "successful login",
			req: LoginRequest{
				Email:    "john.doe@example.com",
				Password: validPassword,
			},
			mockSetup: func(m *mockDB) {
				m.users["john.doe@example.com"] = &User{
					ID:           1,
					Email:        "john.doe@example.com",
					PasswordHash: validHash,
					Active:       true,
				}
			},
			expectedError: "",
		},
		{
			name: "user not found",
			req: LoginRequest{
				Email:    "nonexistent@example.com",
				Password: validPassword,
			},
			mockSetup:     func(m *mockDB) {},
			expectedError: "invalid email or password",
		},
		{
			name: "wrong password",
			req: LoginRequest{
				Email:    "john.doe@example.com",
				Password: "WrongPassword123!",
			},
			mockSetup: func(m *mockDB) {
				m.users["john.doe@example.com"] = &User{
					ID:           1,
					Email:        "john.doe@example.com",
					PasswordHash: validHash,
					Active:       true,
				}
			},
			expectedError: "invalid email or password",
		},
		{
			name: "inactive user",
			req: LoginRequest{
				Email:    "inactive@example.com",
				Password: validPassword,
			},
			mockSetup: func(m *mockDB) {
				m.users["inactive@example.com"] = &User{
					ID:           1,
					Email:        "inactive@example.com",
					PasswordHash: validHash,
					Active:       false,
				}
			},
			expectedError: "user account is inactive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := newMockDB()
			tt.mockSetup(mockDB)

			service := NewService(mockDB)
			user, err := service.Login(tt.req)

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tt.req.Email, user.Email)
				assert.True(t, user.Active)
			}
		})
	}
}
