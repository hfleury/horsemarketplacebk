package media

import (
	"time"

	"github.com/google/uuid"
)

type Media struct {
	ID           uuid.UUID `json:"id" db:"id"`
	FileName     string    `json:"file_name" db:"file_name"`
	OriginalName string    `json:"original_name" db:"original_name"`
	MimeType     string    `json:"mime_type" db:"mime_type"`
	SizeBytes    int64     `json:"size_bytes" db:"size_bytes"`
	URL          string    `json:"url" db:"url"`
	BucketName   string    `json:"bucket_name" db:"bucket_name"`
	Region       string    `json:"region" db:"region"`
	Variants     any       `json:"variants" db:"variants"` // Utilizing 'any' for simplicity with JSONB, or use a specific struct/map
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
