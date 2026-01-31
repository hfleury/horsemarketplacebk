package models

import (
	"encoding/json"

	"github.com/google/uuid"
)

type Horse struct {
	ProductID     uuid.UUID       `json:"product_id"`
	Name          *string         `json:"name"`
	Age           *int            `json:"age"`
	YearOfBirth   *int            `json:"year_of_birth"`
	Gender        *string         `json:"gender"`
	Height        *int            `json:"height"`
	Breed         *string         `json:"breed"`
	Color         *string         `json:"color"`
	DressageLevel *string         `json:"dressage_level"`
	JumpLevel     *string         `json:"jump_level"`
	Orientation   *string         `json:"orientation"`
	Pedigree      json.RawMessage `json:"pedigree"` // JSONB
}

type Vehicle struct {
	ProductID   uuid.UUID `json:"product_id"`
	Make        *string   `json:"make"`
	Model       *string   `json:"model"`
	Year        *int      `json:"year"`
	LoadWeight  *int      `json:"load_weight"`
	TotalWeight *int      `json:"total_weight"`
	Condition   *string   `json:"condition"`
}

type Equipment struct {
	ProductID uuid.UUID `json:"product_id"`
	Make      *string   `json:"make"`
	Model     *string   `json:"model"`
	Size      *string   `json:"size"`
	Condition *string   `json:"condition"`
	SubType   *string   `json:"sub_type"`
	BoomWidth *string   `json:"boom_width"`
}
