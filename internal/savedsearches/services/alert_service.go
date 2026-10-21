package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/common"
	"github.com/hfleury/horsemarketplacebk/internal/email"
	productmodels "github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/models"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/repositories"
)

// listingMatcher is the part of the product repository the alerts need.
type listingMatcher interface {
	FindByID(ctx context.Context, id string) (*productmodels.Product, error)
	MatchesFilter(ctx context.Context, productID, categoryID, query string, filter *productmodels.HorseFilter, locationFilter *productmodels.LocationFilter) (bool, error)
}

// SavedSearchAlertService emails the owners of saved searches that match a
// newly published listing.
type SavedSearchAlertService struct {
	searchRepo  repositories.SavedSearchRepository
	listings    listingMatcher
	sender      email.Sender
	frontendURL string
	logger      config.Logging
}

func NewSavedSearchAlertService(searchRepo repositories.SavedSearchRepository, listings listingMatcher, sender email.Sender, frontendURL string, logger config.Logging) *SavedSearchAlertService {
	return &SavedSearchAlertService{
		searchRepo:  searchRepo,
		listings:    listings,
		sender:      sender,
		frontendURL: frontendURL,
		logger:      logger,
	}
}

// SendMatchAlerts alerts every matching subscription at most once per
// listing. Failed subscriptions are returned joined, so a retry re-attempts
// only those: the rest are already claimed.
func (s *SavedSearchAlertService) SendMatchAlerts(ctx context.Context, productID string) error {
	product, err := s.listings.FindByID(ctx, productID)
	if err != nil {
		return fmt.Errorf("find listing %s: %w", productID, err)
	}
	if product == nil || product.Status != productmodels.StatusPublished {
		s.logger.Log(ctx, config.WarnLevel, "Skipping saved search alerts for a listing that is no longer published", map[string]any{"productId": productID})
		return nil
	}

	subscriptions, err := s.searchRepo.ListAlertSubscriptions(ctx, product.UserID)
	if err != nil {
		return fmt.Errorf("list alert subscriptions: %w", err)
	}

	var alertErrors []error
	for _, subscription := range subscriptions {
		if err := s.alertSubscription(ctx, product, subscription); err != nil {
			alertErrors = append(alertErrors, fmt.Errorf("saved search %s: %w", subscription.Search.ID, err))
		}
	}
	return errors.Join(alertErrors...)
}

func (s *SavedSearchAlertService) alertSubscription(ctx context.Context, product *productmodels.Product, subscription models.AlertSubscription) error {
	search := subscription.Search
	categoryID, query, horseFilter, locationFilter := search.Criteria.ToProductFilters()
	matches, err := s.listings.MatchesFilter(ctx, product.ID.String(), categoryID, query, horseFilter, locationFilter)
	if err != nil {
		return fmt.Errorf("match listing: %w", err)
	}
	if !matches {
		return nil
	}

	claimed, err := s.searchRepo.ClaimAlert(ctx, search.ID, product.ID)
	if err != nil {
		return fmt.Errorf("claim alert: %w", err)
	}
	if !claimed {
		return nil
	}

	subject, body := buildAlertEmail(product, subscription, s.frontendURL)
	if err := s.sender.Send(ctx, subscription.Email, subject, body); err != nil {
		releaseErr := s.searchRepo.ReleaseAlert(ctx, search.ID, product.ID)
		return errors.Join(fmt.Errorf("send alert email: %w", err), releaseErr)
	}

	if err := s.searchRepo.MarkNotified(ctx, search.ID); err != nil {
		s.logger.Log(ctx, config.WarnLevel, "Failed to record saved search notification time", map[string]any{"error": err.Error(), "savedSearchId": search.ID.String()})
	}
	return nil
}

func buildAlertEmail(product *productmodels.Product, subscription models.AlertSubscription, frontendURL string) (subject, body string) {
	subject = fmt.Sprintf("New listing matching %q", subscription.Search.Name)
	body = fmt.Sprintf(
		"%s\n\nA new listing matches your saved search %q.\n\n%s\nPrice: %s\n\nView the listing: %s/listings/%s\n\nManage your alerts: %s/saved-searches",
		alertGreeting(subscription.Username), subscription.Search.Name,
		product.Title, common.FormatPriceSEK(product.PriceSEK),
		frontendURL, product.ID, frontendURL,
	)
	return subject, body
}

func alertGreeting(username string) string {
	if username == "" {
		return "Hi,"
	}
	return fmt.Sprintf("Hi %s,", username)
}
