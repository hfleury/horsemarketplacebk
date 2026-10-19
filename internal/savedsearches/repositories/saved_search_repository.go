package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/db"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/models"
)

type SavedSearchRepository interface {
	Create(ctx context.Context, savedSearch *models.SavedSearch) error
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]models.SavedSearch, error)
	FindByIDForUser(ctx context.Context, id, userID uuid.UUID) (*models.SavedSearch, error)
	Update(ctx context.Context, savedSearch *models.SavedSearch) (*models.SavedSearch, error)
	Delete(ctx context.Context, id, userID uuid.UUID) (bool, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	ListAlertSubscriptions(ctx context.Context, excludeUserID uuid.UUID) ([]models.AlertSubscription, error)
	ClaimAlert(ctx context.Context, savedSearchID, productID uuid.UUID) (bool, error)
	ReleaseAlert(ctx context.Context, savedSearchID, productID uuid.UUID) error
	MarkNotified(ctx context.Context, savedSearchID uuid.UUID) error
}

const savedSearchColumns = `id, user_id, name, criteria, location_label, alerts_enabled, last_notified_at, created_at, updated_at`

var savedSearchColumnsForAlias = "s." + strings.ReplaceAll(savedSearchColumns, ", ", ", s.")

type SavedSearchRepoPsql struct {
	logger config.Logging
	psql   db.Database
}

func NewSavedSearchRepoPsql(psql db.Database, logger config.Logging) *SavedSearchRepoPsql {
	return &SavedSearchRepoPsql{
		psql:   psql,
		logger: logger,
	}
}

func (r *SavedSearchRepoPsql) Create(ctx context.Context, savedSearch *models.SavedSearch) error {
	criteria, err := json.Marshal(savedSearch.Criteria)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO catalog.saved_searches (id, user_id, name, criteria, location_label, alerts_enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at
	`
	err = r.psql.QueryRow(ctx, query,
		savedSearch.ID, savedSearch.UserID, savedSearch.Name, criteria, savedSearch.LocationLabel, savedSearch.AlertsEnabled,
	).Scan(&savedSearch.CreatedAt, &savedSearch.UpdatedAt)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to create saved search", map[string]any{"error": err.Error()})
		return err
	}
	return nil
}

func (r *SavedSearchRepoPsql) ListByUserID(ctx context.Context, userID uuid.UUID) ([]models.SavedSearch, error) {
	query := `SELECT ` + savedSearchColumns + ` FROM catalog.saved_searches WHERE user_id = $1 ORDER BY created_at DESC`
	rows, err := r.psql.Query(ctx, query, userID)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to list saved searches", map[string]any{"error": err.Error()})
		return nil, err
	}
	defer rows.Close()

	savedSearches := []models.SavedSearch{}
	for rows.Next() {
		savedSearch, err := scanSavedSearch(rows)
		if err != nil {
			return nil, err
		}
		savedSearches = append(savedSearches, *savedSearch)
	}
	return savedSearches, rows.Err()
}

// FindByIDForUser returns (nil, nil) when the search doesn't exist or belongs
// to another user, so callers can't tell the two apart.
func (r *SavedSearchRepoPsql) FindByIDForUser(ctx context.Context, id, userID uuid.UUID) (*models.SavedSearch, error) {
	query := `SELECT ` + savedSearchColumns + ` FROM catalog.saved_searches WHERE id = $1 AND user_id = $2`
	return r.findOne(ctx, "Failed to find saved search", query, id, userID)
}

// Update replaces the user-editable fields and returns (nil, nil) when no
// search with that id belongs to the user.
func (r *SavedSearchRepoPsql) Update(ctx context.Context, savedSearch *models.SavedSearch) (*models.SavedSearch, error) {
	criteria, err := json.Marshal(savedSearch.Criteria)
	if err != nil {
		return nil, err
	}

	query := `
		UPDATE catalog.saved_searches
		SET name = $3, criteria = $4, location_label = $5, alerts_enabled = $6, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING ` + savedSearchColumns
	return r.findOne(ctx, "Failed to update saved search", query,
		savedSearch.ID, savedSearch.UserID, savedSearch.Name, criteria, savedSearch.LocationLabel, savedSearch.AlertsEnabled,
	)
}

func (r *SavedSearchRepoPsql) Delete(ctx context.Context, id, userID uuid.UUID) (bool, error) {
	query := `DELETE FROM catalog.saved_searches WHERE id = $1 AND user_id = $2`
	result, err := r.psql.Execute(ctx, query, id, userID)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to delete saved search", map[string]any{"error": err.Error()})
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected > 0, nil
}

func (r *SavedSearchRepoPsql) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM catalog.saved_searches WHERE user_id = $1`
	if err := r.psql.QueryRow(ctx, query, userID).Scan(&count); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to count saved searches", map[string]any{"error": err.Error()})
		return 0, err
	}
	return count, nil
}

// ListAlertSubscriptions returns every alert-enabled search whose owner can
// receive email (active, verified, with an address), except excludeUserID's.
func (r *SavedSearchRepoPsql) ListAlertSubscriptions(ctx context.Context, excludeUserID uuid.UUID) ([]models.AlertSubscription, error) {
	query := `
		SELECT ` + savedSearchColumnsForAlias + `, u.email, COALESCE(u.username, '')
		FROM catalog.saved_searches s
		JOIN auth.users u ON u.id = s.user_id
		WHERE s.alerts_enabled AND s.user_id <> $1
		  AND u.is_active AND u.is_verified AND u.email IS NOT NULL
	`
	rows, err := r.psql.Query(ctx, query, excludeUserID)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to list saved search alert subscriptions", map[string]any{"error": err.Error()})
		return nil, err
	}
	defer rows.Close()

	subscriptions := []models.AlertSubscription{}
	for rows.Next() {
		var subscription models.AlertSubscription
		savedSearch, err := scanSavedSearch(rows, &subscription.Email, &subscription.Username)
		if err != nil {
			return nil, err
		}
		subscription.Search = *savedSearch
		subscriptions = append(subscriptions, subscription)
	}
	return subscriptions, rows.Err()
}

// ClaimAlert records that productID is being alerted to savedSearchID and
// reports false when that pair was already claimed.
func (r *SavedSearchRepoPsql) ClaimAlert(ctx context.Context, savedSearchID, productID uuid.UUID) (bool, error) {
	query := `INSERT INTO catalog.saved_search_alerts (saved_search_id, product_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	result, err := r.psql.Execute(ctx, query, savedSearchID, productID)
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to claim saved search alert", map[string]any{"error": err.Error()})
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected == 1, nil
}

// ReleaseAlert undoes ClaimAlert so a failed alert can be retried.
func (r *SavedSearchRepoPsql) ReleaseAlert(ctx context.Context, savedSearchID, productID uuid.UUID) error {
	query := `DELETE FROM catalog.saved_search_alerts WHERE saved_search_id = $1 AND product_id = $2`
	if _, err := r.psql.Execute(ctx, query, savedSearchID, productID); err != nil {
		r.logger.Log(ctx, config.ErrorLevel, "Failed to release saved search alert", map[string]any{"error": err.Error()})
		return err
	}
	return nil
}

// MarkNotified leaves updated_at alone: it is a system write, not a user edit.
func (r *SavedSearchRepoPsql) MarkNotified(ctx context.Context, savedSearchID uuid.UUID) error {
	query := `UPDATE catalog.saved_searches SET last_notified_at = NOW() WHERE id = $1`
	_, err := r.psql.Execute(ctx, query, savedSearchID)
	return err
}

func (r *SavedSearchRepoPsql) findOne(ctx context.Context, failureMessage, query string, args ...any) (*models.SavedSearch, error) {
	savedSearch, err := scanSavedSearch(r.psql.QueryRow(ctx, query, args...))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		r.logger.Log(ctx, config.ErrorLevel, failureMessage, map[string]any{"error": err.Error()})
		return nil, err
	}
	return savedSearch, nil
}

// scanSavedSearch reads the savedSearchColumns, followed by any extra
// columns the query selects into extraDestinations.
func scanSavedSearch(row interface{ Scan(...any) error }, extraDestinations ...any) (*models.SavedSearch, error) {
	var savedSearch models.SavedSearch
	var criteria []byte
	destinations := append([]any{
		&savedSearch.ID, &savedSearch.UserID, &savedSearch.Name, &criteria, &savedSearch.LocationLabel,
		&savedSearch.AlertsEnabled, &savedSearch.LastNotifiedAt, &savedSearch.CreatedAt, &savedSearch.UpdatedAt,
	}, extraDestinations...)
	if err := row.Scan(destinations...); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(criteria, &savedSearch.Criteria); err != nil {
		return nil, err
	}
	return &savedSearch, nil
}
