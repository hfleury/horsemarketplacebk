package models

import (
	"time"

	"github.com/google/uuid"
)

type HorseAttributeOption struct {
	ID        *uuid.UUID `json:"id"`
	Type      *string    `json:"type"`
	Value     *string    `json:"value"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

type CreateHorseAttributeOptionRequest struct {
	Type  string `json:"type" binding:"required"`
	Value string `json:"value" binding:"required"`
}
