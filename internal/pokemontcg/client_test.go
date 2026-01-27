package pokemontcg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tcgdexModels "github.com/laiambryant/tcgdex/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name   string
		apiKey string
	}{
		{
			name:   "Client with API key",
			apiKey: "test-api-key",
		},
		{
			name:   "Client without API key",
			apiKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.apiKey)

			require.NotNil(t, client)
			assert.NotNil(t, client.sdk)
			assert.NotNil(t, client.limiter)

			// Verify rate limiter is configured correctly
			assert.Equal(t, rate.Every(DefaultRateLimit), client.limiter.Limit())
		})
	}
}

func TestMapTCGDexSetToSet(t *testing.T) {
	tests := []struct {
		name        string
		tcgdexSet   tcgdexModels.SetResume
		expectedSet Set
	}{
		{
			name: "Complete set mapping",
			tcgdexSet: tcgdexModels.SetResume{
				ID:   "base1",
				Name: "Base Set",
				CardCount: tcgdexModels.SetCardCount{
					Official: 102,
					Total:    102,
				},
			},
			expectedSet: Set{
				ID:           "base1",
				Name:         "Base Set",
				PrintedTotal: 102,
				Total:        102,
				PtcgoCode:    "base1",
				UpdatedAt:    "",
			},
		},
		{
			name: "Set with different official and total counts",
			tcgdexSet: tcgdexModels.SetResume{
				ID:   "swsh1",
				Name: "Sword & Shield",
				CardCount: tcgdexModels.SetCardCount{
					Official: 202,
					Total:    216,
				},
			},
			expectedSet: Set{
				ID:           "swsh1",
				Name:         "Sword & Shield",
				PrintedTotal: 202,
				Total:        216,
				PtcgoCode:    "swsh1",
				UpdatedAt:    "",
			},
		},
		{
			name: "Empty set name",
			tcgdexSet: tcgdexModels.SetResume{
				ID:   "test1",
				Name: "",
				CardCount: tcgdexModels.SetCardCount{
					Official: 0,
					Total:    0,
				},
			},
			expectedSet: Set{
				ID:           "test1",
				Name:         "",
				PrintedTotal: 0,
				Total:        0,
				PtcgoCode:    "test1",
				UpdatedAt:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapTCGDexSetToSet(tt.tcgdexSet)
			assert.Equal(t, tt.expectedSet, result)
		})
	}
}

func TestMapTCGDexCardToCard(t *testing.T) {
	illustratorName := "Ken Sugimori"

	tests := []struct {
		name         string
		tcgdexCard   tcgdexModels.Card
		expectedCard Card
	}{
		{
			name: "Complete card mapping with illustrator",
			tcgdexCard: tcgdexModels.Card{
				CardResume: tcgdexModels.CardResume{
					ID:      "base1-25",
					Name:    "Pikachu",
					LocalID: "25",
				},
				Illustrator: &illustratorName,
				Rarity:      "Common",
				Set: tcgdexModels.SetResume{
					ID:   "base1",
					Name: "Base Set",
					CardCount: tcgdexModels.SetCardCount{
						Official: 102,
						Total:    102,
					},
				},
			},
			expectedCard: Card{
				ID:     "base1-25",
				Name:   "Pikachu",
				Number: "25",
				Artist: "Ken Sugimori",
				Rarity: "Common",
				Set: Set{
					ID:           "base1",
					Name:         "Base Set",
					PrintedTotal: 102,
					Total:        102,
				},
				TCGPlayer:  nil,
				CardMarket: nil,
			},
		},
		{
			name: "Card without illustrator",
			tcgdexCard: tcgdexModels.Card{
				CardResume: tcgdexModels.CardResume{
					ID:      "base1-4",
					Name:    "Charizard",
					LocalID: "4",
				},
				Illustrator: nil,
				Rarity:      "Rare Holo",
				Set: tcgdexModels.SetResume{
					ID:   "base1",
					Name: "Base Set",
					CardCount: tcgdexModels.SetCardCount{
						Official: 102,
						Total:    102,
					},
				},
			},
			expectedCard: Card{
				ID:     "base1-4",
				Name:   "Charizard",
				Number: "4",
				Artist: "",
				Rarity: "Rare Holo",
				Set: Set{
					ID:           "base1",
					Name:         "Base Set",
					PrintedTotal: 102,
					Total:        102,
				},
				TCGPlayer:  nil,
				CardMarket: nil,
			},
		},
		{
			name: "Card with empty rarity",
			tcgdexCard: tcgdexModels.Card{
				CardResume: tcgdexModels.CardResume{
					ID:      "promo-1",
					Name:    "Pikachu",
					LocalID: "1",
				},
				Illustrator: nil,
				Rarity:      "",
				Set: tcgdexModels.SetResume{
					ID:   "promo",
					Name: "Promo Cards",
					CardCount: tcgdexModels.SetCardCount{
						Official: 0,
						Total:    0,
					},
				},
			},
			expectedCard: Card{
				ID:     "promo-1",
				Name:   "Pikachu",
				Number: "1",
				Artist: "",
				Rarity: "",
				Set: Set{
					ID:           "promo",
					Name:         "Promo Cards",
					PrintedTotal: 0,
					Total:        0,
				},
				TCGPlayer:  nil,
				CardMarket: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapTCGDexCardToCard(tt.tcgdexCard)
			assert.Equal(t, tt.expectedCard, result)
		})
	}
}

func TestGetStringOrEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{
			name:     "Non-nil string",
			input:    stringPtr("Hello"),
			expected: "Hello",
		},
		{
			name:     "Empty string",
			input:    stringPtr(""),
			expected: "",
		},
		{
			name:     "Nil string",
			input:    nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getStringOrEmpty(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRateLimitedHTTPClient(t *testing.T) {
	t.Run("Rate limiter enforces delay", func(t *testing.T) {
		limiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 1)
		client := &rateLimitedHTTPClient{
			limiter: limiter,
		}

		ctx := context.Background()

		// First call should succeed immediately
		start := time.Now()

		// Consume the first token
		err := client.limiter.Wait(ctx)
		require.NoError(t, err)

		elapsed := time.Since(start)
		assert.Less(t, elapsed, 50*time.Millisecond, "First call should be immediate")

		// Second call should be delayed
		start = time.Now()
		err = client.limiter.Wait(ctx)
		require.NoError(t, err)

		elapsed = time.Since(start)
		assert.GreaterOrEqual(t, elapsed, 90*time.Millisecond, "Second call should be rate limited")
	})

	t.Run("Context cancellation stops rate limiter", func(t *testing.T) {
		limiter := rate.NewLimiter(rate.Every(5*time.Second), 1)
		client := &rateLimitedHTTPClient{
			limiter: limiter,
		}

		// Consume the first token
		err := client.limiter.Wait(context.Background())
		require.NoError(t, err)

		// Create a context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// This should fail immediately due to cancelled context
		start := time.Now()
		err = client.limiter.Wait(ctx)
		elapsed := time.Since(start)

		assert.Error(t, err)
		assert.Less(t, elapsed, 100*time.Millisecond, "Should fail immediately with cancelled context")
	})
}

func TestRateLimitedHTTPClient_Do(t *testing.T) {
	t.Run("API key header is set correctly", func(t *testing.T) {
		// Create a test server that verifies the API key header
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-Api-Key")
			assert.Equal(t, "test-api-key-123", apiKey)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
		}))
		defer server.Close()

		limiter := rate.NewLimiter(rate.Every(10*time.Millisecond), 1)
		client := &rateLimitedHTTPClient{
			client:  &http.Client{Timeout: 5 * time.Second},
			limiter: limiter,
			apiKey:  "test-api-key-123",
		}

		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("No API key header when apiKey is empty", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			apiKey := r.Header.Get("X-Api-Key")
			assert.Empty(t, apiKey)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		limiter := rate.NewLimiter(rate.Every(10*time.Millisecond), 1)
		client := &rateLimitedHTTPClient{
			client:  &http.Client{Timeout: 5 * time.Second},
			limiter: limiter,
			apiKey:  "",
		}

		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Rate limiting is enforced between requests", func(t *testing.T) {
		requestTimes := []time.Time{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestTimes = append(requestTimes, time.Now())
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		limiter := rate.NewLimiter(rate.Every(100*time.Millisecond), 1)
		client := &rateLimitedHTTPClient{
			client:  &http.Client{Timeout: 5 * time.Second},
			limiter: limiter,
			apiKey:  "",
		}

		// Make two requests
		for i := 0; i < 2; i++ {
			req, err := http.NewRequest(http.MethodGet, server.URL, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			resp.Body.Close()
		}

		// Verify rate limiting occurred
		require.Len(t, requestTimes, 2)
		timeDiff := requestTimes[1].Sub(requestTimes[0])
		assert.GreaterOrEqual(t, timeDiff, 90*time.Millisecond, "Second request should be delayed by rate limiter")
	})

	t.Run("Context cancellation returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		// Use a slow rate limiter
		limiter := rate.NewLimiter(rate.Every(5*time.Second), 1)
		client := &rateLimitedHTTPClient{
			client:  &http.Client{Timeout: 5 * time.Second},
			limiter: limiter,
			apiKey:  "",
		}

		// Consume the first token
		err := client.limiter.Wait(context.Background())
		require.NoError(t, err)

		// Create a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		start := time.Now()
		_, err = client.Do(req)
		elapsed := time.Since(start)

		assert.Error(t, err)
		assert.Less(t, elapsed, 100*time.Millisecond, "Should fail quickly with cancelled context")
	})

	t.Run("HTTP errors are propagated", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal server error"}`))
		}))
		defer server.Close()

		limiter := rate.NewLimiter(rate.Every(10*time.Millisecond), 1)
		client := &rateLimitedHTTPClient{
			client:  &http.Client{Timeout: 5 * time.Second},
			limiter: limiter,
			apiKey:  "",
		}

		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("Network errors are returned", func(t *testing.T) {
		limiter := rate.NewLimiter(rate.Every(10*time.Millisecond), 1)
		client := &rateLimitedHTTPClient{
			client:  &http.Client{Timeout: 5 * time.Second},
			limiter: limiter,
			apiKey:  "",
		}

		// Request to invalid URL
		req, err := http.NewRequest(http.MethodGet, "http://localhost:0", nil)
		require.NoError(t, err)

		_, err = client.Do(req)
		assert.Error(t, err, "Should return error for invalid URL")
	})
}

func TestPaginatedResponse(t *testing.T) {
	t.Run("PaginatedResponse structure", func(t *testing.T) {
		resp := &PaginatedResponse{
			Page:       1,
			PageSize:   250,
			TotalCount: 100,
		}

		assert.Equal(t, 1, resp.Page)
		assert.Equal(t, 250, resp.PageSize)
		assert.Equal(t, 100, resp.TotalCount)
	})
}

func TestConstants(t *testing.T) {
	t.Run("Default values are set correctly", func(t *testing.T) {
		assert.Equal(t, 250, DefaultPageSize)
		assert.Equal(t, 250, MaxPageSize)
		assert.Equal(t, 30*time.Second, DefaultTimeout)
		assert.Equal(t, 500*time.Millisecond, DefaultRateLimit)
	})
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
