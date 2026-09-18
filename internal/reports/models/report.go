package models

import (
	"time"

	"github.com/google/uuid"
)

type ReportReason string

type ReportStatus string

const (
	ReasonSpam          ReportReason = "spam"
	ReasonFraud         ReportReason = "fraud"
	ReasonInappropriate ReportReason = "inappropriate"
	ReasonDuplicate     ReportReason = "duplicate"
	ReasonOther         ReportReason = "other"

	StatusPending   ReportStatus = "pending"
	StatusReviewed  ReportStatus = "reviewed"
	StatusDismissed ReportStatus = "dismissed"
)

type Report struct {
	ID             uuid.UUID    `json:"id"`
	ProductID      uuid.UUID    `json:"product_id"`
	ReporterUserID uuid.UUID    `json:"reporter_user_id"`
	Reason         ReportReason `json:"reason"`
	Description    *string      `json:"description"`
	Status         ReportStatus `json:"status"`
	ReviewedBy     *uuid.UUID   `json:"reviewed_by"`
	ReviewedAt     *time.Time   `json:"reviewed_at"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

type CreateReportRequest struct {
	ProductID   string  `json:"product_id" binding:"required"`
	Reason      string  `json:"reason" binding:"required"`
	Description *string `json:"description"`
}

type ReviewReportRequest struct {
	Status string `json:"status" binding:"required"`
}
