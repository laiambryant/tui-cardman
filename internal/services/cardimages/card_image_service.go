package cardimages

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CardImageService defines the interface for card image-related operations
type CardImageService interface {
	ReplaceCardImages(ctx context.Context, tx *sql.Tx, cardID int64, smallURL, largeURL string) error
	DeleteCardImages(ctx context.Context, tx *sql.Tx, cardID int64) error
}

// CardImageServiceImpl implements the CardImageService interface
type CardImageServiceImpl struct {
	db *sql.DB
}

// NewCardImageService creates a new instance of CardImageServiceImpl
func NewCardImageService(db *sql.DB) CardImageService {
	return &CardImageServiceImpl{db: db}
}

const (
	deleteCardImagesQuery = `DELETE FROM card_images WHERE card_id = ?`
	insertCardImagesQuery = `INSERT INTO card_images (card_id, small_url, large_url, updated_at)
	    VALUES (?, ?, ?, ?)`
)

// DeleteCardImages deletes all images for a specific card within a transaction
func (s *CardImageServiceImpl) DeleteCardImages(ctx context.Context, tx *sql.Tx, cardID int64) error {
	if _, err := tx.ExecContext(ctx, deleteCardImagesQuery, cardID); err != nil {
		return fmt.Errorf("failed to delete card images: %w", err)
	}
	return nil
}

// ReplaceCardImages deletes old images and inserts new ones for a card within a transaction
func (s *CardImageServiceImpl) ReplaceCardImages(ctx context.Context, tx *sql.Tx, cardID int64, smallURL, largeURL string) error {
	if err := s.DeleteCardImages(ctx, tx, cardID); err != nil {
		return err
	}

	if smallURL == "" && largeURL == "" {
		return nil
	}

	if _, err := tx.ExecContext(ctx, insertCardImagesQuery, cardID, smallURL, largeURL, time.Now()); err != nil {
		return fmt.Errorf("failed to insert card images: %w", err)
	}
	return nil
}
