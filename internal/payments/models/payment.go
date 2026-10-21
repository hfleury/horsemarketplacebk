package models

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// PaymentStatus mirrors the status CHECK on catalog.payments.
type PaymentStatus string

const (
	StatusPending  PaymentStatus = "pending"
	StatusPaid     PaymentStatus = "paid"
	StatusFailed   PaymentStatus = "failed"
	StatusRefunded PaymentStatus = "refunded"
)

// Payment is one charge for one boost. UserID is nil once the buyer's account
// has been deleted, because payment records are kept for bookkeeping.
type Payment struct {
	ID                uuid.UUID     `json:"id"`
	UserID            *uuid.UUID    `json:"user_id"`
	ProviderID        uuid.UUID     `json:"provider_id"`
	ProviderReference *string       `json:"provider_reference"`
	AmountSEK         float64       `json:"amount_sek"`
	Status            PaymentStatus `json:"status"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

// ProductBoost is the window during which a paid boost shows on a listing.
type ProductBoost struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"product_id"`
	PaymentID uuid.UUID `json:"payment_id"`
	Tier      string    `json:"tier"`
	StartsAt  time.Time `json:"starts_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CheckoutRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	TierCode  string `json:"tier_code" binding:"required"`
	Provider  string `json:"provider" binding:"required"`
	// StartsAt is optional; nil or a past time means "start now".
	StartsAt *time.Time `json:"starts_at"`
}

type CheckoutResponse struct {
	PaymentID   uuid.UUID `json:"payment_id"`
	CheckoutURL string    `json:"checkout_url"`
}

type PaymentStatusResponse struct {
	PaymentID uuid.UUID     `json:"payment_id"`
	Status    PaymentStatus `json:"status"`
	Tier      string        `json:"tier"`
	ProductID uuid.UUID     `json:"product_id"`
	StartsAt  time.Time     `json:"starts_at"`
	ExpiresAt time.Time     `json:"expires_at"`
}

// ConfirmedPayment is a payment that has just become paid, with its boost
// window already shifted to start no earlier than the confirmation.
type ConfirmedPayment struct {
	PaymentID uuid.UUID
	UserID    *uuid.UUID
	AmountSEK float64
	Boost     ProductBoost
}

// SEKToOre converts a SEK amount to the integer öre that provider APIs expect.
// It rounds because amounts like 99.99 are slightly below their value as floats.
func SEKToOre(amountSEK float64) int64 {
	return int64(math.Round(amountSEK * 100))
}
