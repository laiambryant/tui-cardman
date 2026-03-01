package db

import (
	"testing"

	"github.com/laiambryant/gotestutils/ctesting"
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
