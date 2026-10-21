package repositories

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/payments/models"
)

type PaymentRepository interface {
	// FindProviderIDByCode returns nil when no provider row has that code.
	FindProviderIDByCode(ctx context.Context, code string) (*uuid.UUID, error)
	// CreatePendingWithBoost inserts both rows atomically: neither exists without the other.
	CreatePendingWithBoost(ctx context.Context, payment *models.Payment, boost *models.ProductBoost) error
	SetProviderReference(ctx context.Context, paymentID uuid.UUID, reference string) error
	// MarkCheckoutFailed fails a payment whose checkout could not be created.
	MarkCheckoutFailed(ctx context.Context, paymentID uuid.UUID) error
	// ConfirmPaid moves a pending payment to paid and shifts its boost window to
	// start no earlier than now. It returns nil when there is no pending payment
	// with that reference, i.e. it was already applied or never existed.
	ConfirmPaid(ctx context.Context, providerID uuid.UUID, reference string) (*models.ConfirmedPayment, error)
	// TransitionStatus reports false when the payment isn't currently in `from`.
	TransitionStatus(ctx context.Context, providerID uuid.UUID, reference string, from, to models.PaymentStatus) (bool, error)
	PaymentExists(ctx context.Context, providerID uuid.UUID, reference string) (bool, error)
	// FindStatusForUser returns nil when the payment doesn't exist or belongs to
	// another user, so callers can't tell the two apart.
	FindStatusForUser(ctx context.Context, paymentID, userID uuid.UUID) (*models.PaymentStatusResponse, error)
}

type PaymentRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewPaymentRepoPsql(psql db.Database, logger config.Logging) *PaymentRepoPsql {
	return &PaymentRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

func (r *PaymentRepoPsql) FindProviderIDByCode(ctx context.Context, code string) (*uuid.UUID, error) {
	var providerID uuid.UUID
	query := `SELECT id FROM catalog.payment_providers WHERE code = $1`
	err := r.psql.QueryRow(ctx, query, code).Scan(&providerID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to find payment provider", map[string]any{"error": err.Error()})
		return nil, err
	}
	return &providerID, nil
}

func (r *PaymentRepoPsql) CreatePendingWithBoost(ctx context.Context, payment *models.Payment, boost *models.ProductBoost) error {
	tx, err := r.psql.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := insertPayment(ctx, tx, payment); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to insert payment", map[string]any{"error": err.Error()})
		return err
	}
	if err := insertBoost(ctx, tx, boost); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to insert product boost", map[string]any{"error": err.Error()})
		return err
	}
	return tx.Commit()
}

func insertPayment(ctx context.Context, tx *sql.Tx, payment *models.Payment) error {
	query := `
		INSERT INTO catalog.payments (id, user_id, provider_id, amount_sek, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`
	return tx.QueryRowContext(ctx, query,
		payment.ID, payment.UserID, payment.ProviderID, payment.AmountSEK, payment.Status,
	).Scan(&payment.CreatedAt, &payment.UpdatedAt)
}

func insertBoost(ctx context.Context, tx *sql.Tx, boost *models.ProductBoost) error {
	query := `
		INSERT INTO catalog.product_boosts (id, product_id, payment_id, tier, starts_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := tx.ExecContext(ctx, query,
		boost.ID, boost.ProductID, boost.PaymentID, boost.Tier, boost.StartsAt, boost.ExpiresAt,
	)
	return err
}

func (r *PaymentRepoPsql) SetProviderReference(ctx context.Context, paymentID uuid.UUID, reference string) error {
	query := `UPDATE catalog.payments SET provider_reference = $2, updated_at = NOW() WHERE id = $1`
	if _, err := r.psql.Execute(ctx, query, paymentID, reference); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to set payment provider reference", map[string]any{"error": err.Error()})
		return err
	}
	return nil
}

func (r *PaymentRepoPsql) MarkCheckoutFailed(ctx context.Context, paymentID uuid.UUID) error {
	query := `UPDATE catalog.payments SET status = 'failed', updated_at = NOW() WHERE id = $1 AND status = 'pending'`
	if _, err := r.psql.Execute(ctx, query, paymentID); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to mark checkout failed", map[string]any{"error": err.Error()})
		return err
	}
	return nil
}

// ConfirmPaid's `status = 'pending'` guard is the claim: concurrent duplicate
// deliveries serialize on the row lock, and only the first one gets a row.
func (r *PaymentRepoPsql) ConfirmPaid(ctx context.Context, providerID uuid.UUID, reference string) (*models.ConfirmedPayment, error) {
	tx, err := r.psql.BeginTransaction(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	confirmed, err := markPaid(ctx, tx, providerID, reference)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to mark payment paid", map[string]any{"error": err.Error()})
		return nil, err
	}
	if err := shiftBoostWindow(ctx, tx, &confirmed.Boost); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to shift boost window", map[string]any{"error": err.Error()})
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return confirmed, nil
}

func markPaid(ctx context.Context, tx *sql.Tx, providerID uuid.UUID, reference string) (*models.ConfirmedPayment, error) {
	query := `
		UPDATE catalog.payments SET status = 'paid', updated_at = NOW()
		WHERE provider_id = $1 AND provider_reference = $2 AND status = 'pending'
		RETURNING id, user_id, amount_sek
	`
	var confirmed models.ConfirmedPayment
	err := tx.QueryRowContext(ctx, query, providerID, reference).
		Scan(&confirmed.PaymentID, &confirmed.UserID, &confirmed.AmountSEK)
	if err != nil {
		return nil, err
	}
	return &confirmed, nil
}

// shiftBoostWindow keeps the purchased duration: Postgres evaluates every SET
// expression against the old row, so expires_at - starts_at is the original length.
func shiftBoostWindow(ctx context.Context, tx *sql.Tx, boost *models.ProductBoost) error {
	query := `
		UPDATE catalog.product_boosts
		SET starts_at  = GREATEST(starts_at, NOW()),
		    expires_at = GREATEST(starts_at, NOW()) + (expires_at - starts_at),
		    updated_at = NOW()
		WHERE payment_id = $1
		RETURNING id, product_id, payment_id, tier, starts_at, expires_at
	`
	return tx.QueryRowContext(ctx, query, boost.PaymentID).Scan(
		&boost.ID, &boost.ProductID, &boost.PaymentID, &boost.Tier, &boost.StartsAt, &boost.ExpiresAt,
	)
}

func (r *PaymentRepoPsql) TransitionStatus(ctx context.Context, providerID uuid.UUID, reference string, from, to models.PaymentStatus) (bool, error) {
	query := `
		UPDATE catalog.payments SET status = $4, updated_at = NOW()
		WHERE provider_id = $1 AND provider_reference = $2 AND status = $3
	`
	result, err := r.psql.Execute(ctx, query, providerID, reference, from, to)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to transition payment status", map[string]any{"error": err.Error()})
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected == 1, nil
}

func (r *PaymentRepoPsql) PaymentExists(ctx context.Context, providerID uuid.UUID, reference string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM catalog.payments WHERE provider_id = $1 AND provider_reference = $2)`
	if err := r.psql.QueryRow(ctx, query, providerID, reference).Scan(&exists); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to check payment exists", map[string]any{"error": err.Error()})
		return false, err
	}
	return exists, nil
}

func (r *PaymentRepoPsql) FindStatusForUser(ctx context.Context, paymentID, userID uuid.UUID) (*models.PaymentStatusResponse, error) {
	query := `
		SELECT p.id, p.status, b.tier, b.product_id, b.starts_at, b.expires_at
		FROM catalog.payments p
		JOIN catalog.product_boosts b ON b.payment_id = p.id
		WHERE p.id = $1 AND p.user_id = $2
	`
	var status models.PaymentStatusResponse
	err := r.psql.QueryRow(ctx, query, paymentID, userID).Scan(
		&status.PaymentID, &status.Status, &status.Tier, &status.ProductID, &status.StartsAt, &status.ExpiresAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to find payment status", map[string]any{"error": err.Error()})
		return nil, err
	}
	return &status, nil
}
