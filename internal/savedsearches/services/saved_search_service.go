package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/models"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/repositories"
)

var (
	ErrInvalidSavedSearchID    = errors.New("invalid saved search id")
	ErrInvalidName             = errors.New("name must be between 1 and 100 characters")
	ErrInvalidCriteria         = errors.New("invalid search criteria")
	ErrSavedSearchNotFound     = errors.New("saved search not found")
	ErrSavedSearchLimitReached = errors.New("saved search limit reached")
)

const (
	maxSavedSearchesPerUser = 20
	maxNameLength           = 100
	maxLocationLabelLength  = 100
	minRadiusKm             = 10
	maxRadiusKm             = 500
)

type SavedSearchService struct {
	repo   repositories.SavedSearchRepository
	logger config.Logging
}

func NewSavedSearchService(repo repositories.SavedSearchRepository, logger config.Logging) *SavedSearchService {
	return &SavedSearchService{
		repo:   repo,
		logger: logger,
	}
}

func (s *SavedSearchService) Create(ctx context.Context, userID string, req models.SavedSearchRequest) (*models.SavedSearch, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	savedSearch, err := buildSavedSearch(uuid.New(), userUUID, req)
	if err != nil {
		return nil, err
	}

	count, err := s.repo.CountByUserID(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	if count >= maxSavedSearchesPerUser {
		return nil, ErrSavedSearchLimitReached
	}

	if err := s.repo.Create(ctx, savedSearch); err != nil {
		return nil, err
	}
	return savedSearch, nil
}

func (s *SavedSearchService) List(ctx context.Context, userID string) ([]models.SavedSearch, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	return s.repo.ListByUserID(ctx, userUUID)
}

func (s *SavedSearchService) Get(ctx context.Context, userID, id string) (*models.SavedSearch, error) {
	userUUID, savedSearchUUID, err := parseIDs(userID, id)
	if err != nil {
		return nil, err
	}

	savedSearch, err := s.repo.FindByIDForUser(ctx, savedSearchUUID, userUUID)
	if err != nil {
		return nil, err
	}
	if savedSearch == nil {
		return nil, ErrSavedSearchNotFound
	}
	return savedSearch, nil
}

// Update is a full replace: omitted optional fields fall back to their defaults.
func (s *SavedSearchService) Update(ctx context.Context, userID, id string, req models.SavedSearchRequest) (*models.SavedSearch, error) {
	userUUID, savedSearchUUID, err := parseIDs(userID, id)
	if err != nil {
		return nil, err
	}

	savedSearch, err := buildSavedSearch(savedSearchUUID, userUUID, req)
	if err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, savedSearch)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrSavedSearchNotFound
	}
	return updated, nil
}

func (s *SavedSearchService) Delete(ctx context.Context, userID, id string) error {
	userUUID, savedSearchUUID, err := parseIDs(userID, id)
	if err != nil {
		return err
	}

	deleted, err := s.repo.Delete(ctx, savedSearchUUID, userUUID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrSavedSearchNotFound
	}
	return nil
}

func parseIDs(userID, id string) (userUUID, savedSearchUUID uuid.UUID, err error) {
	userUUID, err = uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	savedSearchUUID, err = uuid.Parse(id)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidSavedSearchID
	}
	return userUUID, savedSearchUUID, nil
}

func buildSavedSearch(id, userID uuid.UUID, req models.SavedSearchRequest) (*models.SavedSearch, error) {
	name, err := normalizeName(req.Name)
	if err != nil {
		return nil, err
	}
	criteria, err := normalizeCriteria(*req.Criteria)
	if err != nil {
		return nil, err
	}
	locationLabel, err := normalizeLocationLabel(req.LocationLabel, criteria)
	if err != nil {
		return nil, err
	}

	alertsEnabled := true
	if req.AlertsEnabled != nil {
		alertsEnabled = *req.AlertsEnabled
	}

	return &models.SavedSearch{
		ID:            id,
		UserID:        userID,
		Name:          name,
		Criteria:      criteria,
		LocationLabel: locationLabel,
		AlertsEnabled: alertsEnabled,
	}, nil
}

func normalizeName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	length := utf8.RuneCountInString(trimmed)
	if length == 0 || length > maxNameLength {
		return "", ErrInvalidName
	}
	return trimmed, nil
}

func normalizeLocationLabel(label *string, criteria models.SearchCriteria) (*string, error) {
	trimmed := trimToNil(label)
	if trimmed == nil {
		return nil, nil
	}
	if criteria.Lat == nil {
		return nil, fmt.Errorf("%w: location_label requires lat, lng and radius_km", ErrInvalidCriteria)
	}
	if utf8.RuneCountInString(*trimmed) > maxLocationLabelLength {
		return nil, fmt.Errorf("%w: location_label must be at most %d characters", ErrInvalidCriteria, maxLocationLabelLength)
	}
	return trimmed, nil
}

func normalizeCriteria(criteria models.SearchCriteria) (models.SearchCriteria, error) {
	criteria.Query = trimToNil(criteria.Query)
	criteria.CategoryID = trimToNil(criteria.CategoryID)
	criteria.Breed = trimToNil(criteria.Breed)
	criteria.Gender = trimToNil(criteria.Gender)
	criteria.Discipline = trimToNil(criteria.Discipline)

	validators := []func(models.SearchCriteria) error{
		validateCategoryID,
		validateRanges,
		validateLocation,
		validateNotEmpty,
	}
	for _, validate := range validators {
		if err := validate(criteria); err != nil {
			return models.SearchCriteria{}, err
		}
	}
	return criteria, nil
}

func validateCategoryID(criteria models.SearchCriteria) error {
	if criteria.CategoryID == nil {
		return nil
	}
	if _, err := uuid.Parse(*criteria.CategoryID); err != nil {
		return fmt.Errorf("%w: category_id must be a valid id", ErrInvalidCriteria)
	}
	return nil
}

func validateRanges(criteria models.SearchCriteria) error {
	if err := validateRange("age", criteria.MinAge, criteria.MaxAge); err != nil {
		return err
	}
	if err := validateRange("height", criteria.MinHeight, criteria.MaxHeight); err != nil {
		return err
	}
	return validateRange("price", criteria.MinPrice, criteria.MaxPrice)
}

func validateRange[T int | float64](field string, minimum, maximum *T) error {
	if (minimum != nil && *minimum < 0) || (maximum != nil && *maximum < 0) {
		return fmt.Errorf("%w: %s must not be negative", ErrInvalidCriteria, field)
	}
	if minimum != nil && maximum != nil && *minimum > *maximum {
		return fmt.Errorf("%w: min_%s exceeds max_%s", ErrInvalidCriteria, field, field)
	}
	return nil
}

// validateLocation mirrors the listing search's rules for lat, lng and radius_km.
func validateLocation(criteria models.SearchCriteria) error {
	if criteria.Lat == nil && criteria.Lng == nil && criteria.RadiusKm == nil {
		return nil
	}
	if criteria.Lat == nil || criteria.Lng == nil || criteria.RadiusKm == nil {
		return fmt.Errorf("%w: lat, lng and radius_km must be provided together", ErrInvalidCriteria)
	}
	if *criteria.Lat < -90 || *criteria.Lat > 90 || *criteria.Lng < -180 || *criteria.Lng > 180 {
		return fmt.Errorf("%w: lat or lng is out of range", ErrInvalidCriteria)
	}
	if *criteria.RadiusKm < minRadiusKm || *criteria.RadiusKm > maxRadiusKm {
		return fmt.Errorf("%w: radius_km must be between %d and %d", ErrInvalidCriteria, minRadiusKm, maxRadiusKm)
	}
	return nil
}

func validateNotEmpty(criteria models.SearchCriteria) error {
	if criteria == (models.SearchCriteria{}) {
		return fmt.Errorf("%w: at least one criterion is required", ErrInvalidCriteria)
	}
	return nil
}

func trimToNil(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
