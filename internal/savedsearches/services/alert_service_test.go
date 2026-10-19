package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	mockconfig "github.com/hfleury/horsemarketplacebk/internal/mocks/config"
	mockemail "github.com/hfleury/horsemarketplacebk/internal/mocks/email"
	mockProducts "github.com/hfleury/horsemarketplacebk/internal/mocks/products"
	mocksavedsearches "github.com/hfleury/horsemarketplacebk/internal/mocks/savedsearches"
	productmodels "github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/models"
	"github.com/hfleury/horsemarketplacebk/internal/savedsearches/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const alertFrontendURL = "https://horses.example"

type alertTestFixture struct {
	service    *services.SavedSearchAlertService
	searchRepo *mocksavedsearches.MockSavedSearchRepository
	listings   *mockProducts.MockProductRepo
	sender     *mockemail.MockSender
	listing    *productmodels.Product
}

func newAlertTestFixture(logger config.Logging) alertTestFixture {
	fixture := alertTestFixture{
		searchRepo: new(mocksavedsearches.MockSavedSearchRepository),
		listings:   new(mockProducts.MockProductRepo),
		sender:     mockemail.NewMockSender(),
		listing: &productmodels.Product{
			ID:       uuid.New(),
			UserID:   uuid.New(),
			Status:   productmodels.StatusPublished,
			Title:    "Gentle Arabian mare",
			PriceSEK: floatPtr(12500),
		},
	}
	fixture.service = services.NewSavedSearchAlertService(fixture.searchRepo, fixture.listings, fixture.sender, alertFrontendURL, logger)
	return fixture
}

func (f alertTestFixture) listingID() string { return f.listing.ID.String() }

func (f alertTestFixture) expectListingFound() {
	f.listings.On("FindByID", mock.Anything, f.listingID()).Return(f.listing, nil)
}

func (f alertTestFixture) expectSubscriptions(subscriptions ...models.AlertSubscription) {
	f.searchRepo.On("ListAlertSubscriptions", mock.Anything, f.listing.UserID).Return(subscriptions, nil)
}

func (f alertTestFixture) expectMatch(subscription models.AlertSubscription, matches bool, err error) {
	categoryID, query, horseFilter, locationFilter := subscription.Search.Criteria.ToProductFilters()
	f.listings.On("MatchesFilter", mock.Anything, f.listingID(), categoryID, query, horseFilter, locationFilter).Return(matches, err).Once()
}

func (f alertTestFixture) expectClaim(subscription models.AlertSubscription, claimed bool) {
	f.searchRepo.On("ClaimAlert", mock.Anything, subscription.Search.ID, f.listing.ID).Return(claimed, nil)
}

func (f alertTestFixture) expectMarkNotified(subscription models.AlertSubscription, err error) {
	f.searchRepo.On("MarkNotified", mock.Anything, subscription.Search.ID).Return(err)
}

// expectAlertDelivered sets up a subscription that matches, is claimed, and is
// marked notified after its email goes out.
func (f alertTestFixture) expectAlertDelivered(subscription models.AlertSubscription) {
	f.expectMatch(subscription, true, nil)
	f.expectClaim(subscription, true)
	f.expectMarkNotified(subscription, nil)
}

func newSubscription(name, emailAddress, username string, criteria models.SearchCriteria) models.AlertSubscription {
	return models.AlertSubscription{
		Search:   models.SavedSearch{ID: uuid.New(), UserID: uuid.New(), Name: name, Criteria: criteria, AlertsEnabled: true},
		Email:    emailAddress,
		Username: username,
	}
}

func TestSavedSearchAlertService_SendMatchAlerts_ListingNotFound_SkipsSubscriptionLookup(t *testing.T) {
	fixture := newAlertTestFixture(config.NewZerologService())
	fixture.listings.On("FindByID", mock.Anything, fixture.listingID()).Return(nil, nil)

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	assert.NoError(t, err)
	fixture.searchRepo.AssertNotCalled(t, "ListAlertSubscriptions", mock.Anything, mock.Anything)
}

func TestSavedSearchAlertService_SendMatchAlerts_ListingNoLongerPublished_SkipsSubscriptionLookup(t *testing.T) {
	fixture := newAlertTestFixture(config.NewZerologService())
	fixture.listing.Status = productmodels.StatusSold
	fixture.expectListingFound()

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	assert.NoError(t, err)
	fixture.searchRepo.AssertNotCalled(t, "ListAlertSubscriptions", mock.Anything, mock.Anything)
}

func TestSavedSearchAlertService_SendMatchAlerts_FindListingError_IsReturned(t *testing.T) {
	fixture := newAlertTestFixture(config.NewZerologService())
	dbDown := errors.New("db down")
	fixture.listings.On("FindByID", mock.Anything, fixture.listingID()).Return(nil, dbDown)

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	assert.ErrorIs(t, err, dbDown)
}

func TestSavedSearchAlertService_SendMatchAlerts_ListSubscriptionsError_IsReturned(t *testing.T) {
	fixture := newAlertTestFixture(config.NewZerologService())
	dbDown := errors.New("db down")
	fixture.expectListingFound()
	fixture.searchRepo.On("ListAlertSubscriptions", mock.Anything, fixture.listing.UserID).Return(nil, dbDown)

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	assert.ErrorIs(t, err, dbDown)
}

func TestSavedSearchAlertService_SendMatchAlerts_NoSubscriptions_SendsNothing(t *testing.T) {
	fixture := newAlertTestFixture(config.NewZerologService())
	fixture.expectListingFound()
	fixture.expectSubscriptions()

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	assert.NoError(t, err)
	assert.Empty(t, fixture.sender.Sent)
}

func TestSavedSearchAlertService_SendMatchAlerts_NonMatchingSearch_IsNotClaimedOrEmailed(t *testing.T) {
	fixture := newAlertTestFixture(config.NewZerologService())
	subscription := newSubscription("Ponies", "buyer@example.com", "buyer", models.SearchCriteria{Breed: stringPtr("Shetland")})
	fixture.expectListingFound()
	fixture.expectSubscriptions(subscription)
	fixture.expectMatch(subscription, false, nil)

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	assert.NoError(t, err)
	fixture.searchRepo.AssertNotCalled(t, "ClaimAlert", mock.Anything, mock.Anything, mock.Anything)
	assert.Empty(t, fixture.sender.Sent)
}

func TestSavedSearchAlertService_SendMatchAlerts_MatchingSearch_EmailsSubscriber(t *testing.T) {
	fixture := newAlertTestFixture(config.NewZerologService())
	subscription := newSubscription("Arabians", "buyer@example.com", "anna", models.SearchCriteria{Breed: stringPtr("Arabian")})
	fixture.expectListingFound()
	fixture.expectSubscriptions(subscription)
	fixture.expectAlertDelivered(subscription)

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	require.NoError(t, err)
	require.Len(t, fixture.sender.Sent, 1)
	sent := fixture.sender.Sent[0]
	assert.Equal(t, "buyer@example.com", sent.To)
	assert.Contains(t, sent.Subject, `"Arabians"`)
	assert.Contains(t, sent.Body, "Hi anna,")
	assert.Contains(t, sent.Body, "Gentle Arabian mare")
	assert.Contains(t, sent.Body, "12500 SEK")
	assert.Contains(t, sent.Body, alertFrontendURL+"/listings/"+fixture.listingID())
	assert.Contains(t, sent.Body, alertFrontendURL+"/saved-searches")
	fixture.searchRepo.AssertCalled(t, "MarkNotified", mock.Anything, subscription.Search.ID)
}

func TestSavedSearchAlertService_SendMatchAlerts_AlreadyClaimed_IsNotEmailedAgain(t *testing.T) {
	fixture := newAlertTestFixture(config.NewZerologService())
	subscription := newSubscription("Arabians", "buyer@example.com", "anna", models.SearchCriteria{})
	fixture.expectListingFound()
	fixture.expectSubscriptions(subscription)
	fixture.expectMatch(subscription, true, nil)
	fixture.expectClaim(subscription, false)

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	assert.NoError(t, err)
	assert.Empty(t, fixture.sender.Sent)
	fixture.searchRepo.AssertNotCalled(t, "MarkNotified", mock.Anything, mock.Anything)
}

func TestSavedSearchAlertService_SendMatchAlerts_SendFailure_ReleasesClaimAndContinues(t *testing.T) {
	fixture := newAlertTestFixture(config.NewZerologService())
	smtpDown := errors.New("smtp down")
	failing := newSubscription("Mares", "failing@example.com", "", models.SearchCriteria{Gender: stringPtr("mare")})
	delivered := newSubscription("Arabians", "buyer@example.com", "anna", models.SearchCriteria{Breed: stringPtr("Arabian")})
	fixture.sender.ErrFor = map[string]error{"failing@example.com": smtpDown}
	fixture.expectListingFound()
	fixture.expectSubscriptions(failing, delivered)
	fixture.expectMatch(failing, true, nil)
	fixture.expectClaim(failing, true)
	fixture.searchRepo.On("ReleaseAlert", mock.Anything, failing.Search.ID, fixture.listing.ID).Return(nil)
	fixture.expectAlertDelivered(delivered)

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	assert.ErrorIs(t, err, smtpDown)
	fixture.searchRepo.AssertCalled(t, "ReleaseAlert", mock.Anything, failing.Search.ID, fixture.listing.ID)
	fixture.searchRepo.AssertNotCalled(t, "MarkNotified", mock.Anything, failing.Search.ID)
	require.Len(t, fixture.sender.Sent, 1)
	assert.Equal(t, "buyer@example.com", fixture.sender.Sent[0].To)
}

func TestSavedSearchAlertService_SendMatchAlerts_MatchError_OtherSubscriptionsStillProcessed(t *testing.T) {
	fixture := newAlertTestFixture(config.NewZerologService())
	queryFailed := errors.New("query failed")
	broken := newSubscription("Mares", "other@example.com", "", models.SearchCriteria{Gender: stringPtr("mare")})
	delivered := newSubscription("Arabians", "buyer@example.com", "anna", models.SearchCriteria{Breed: stringPtr("Arabian")})
	fixture.expectListingFound()
	fixture.expectSubscriptions(broken, delivered)
	fixture.expectMatch(broken, false, queryFailed)
	fixture.expectAlertDelivered(delivered)

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	assert.ErrorIs(t, err, queryFailed)
	require.Len(t, fixture.sender.Sent, 1)
	assert.Equal(t, "buyer@example.com", fixture.sender.Sent[0].To)
}

func TestSavedSearchAlertService_SendMatchAlerts_MarkNotifiedError_IsOnlyLogged(t *testing.T) {
	ctrl := gomock.NewController(t)
	logger := mockconfig.NewMockLogging(ctrl)
	logger.EXPECT().Log(gomock.Any(), config.WarnLevel, "Failed to record saved search notification time", gomock.Any()).Times(1)
	fixture := newAlertTestFixture(logger)
	subscription := newSubscription("Arabians", "buyer@example.com", "anna", models.SearchCriteria{})
	fixture.expectListingFound()
	fixture.expectSubscriptions(subscription)
	fixture.expectMatch(subscription, true, nil)
	fixture.expectClaim(subscription, true)
	fixture.expectMarkNotified(subscription, errors.New("db down"))

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	assert.NoError(t, err)
	assert.Len(t, fixture.sender.Sent, 1)
}

func TestSavedSearchAlertService_SendMatchAlerts_NoPriceAndNoUsername_UsesFallbackWording(t *testing.T) {
	fixture := newAlertTestFixture(config.NewZerologService())
	fixture.listing.PriceSEK = nil
	subscription := newSubscription("Anything", "buyer@example.com", "", models.SearchCriteria{})
	fixture.expectListingFound()
	fixture.expectSubscriptions(subscription)
	fixture.expectAlertDelivered(subscription)

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	require.NoError(t, err)
	require.Len(t, fixture.sender.Sent, 1)
	assert.Contains(t, fixture.sender.Sent[0].Body, "Price on request")
	assert.Contains(t, fixture.sender.Sent[0].Body, "Hi,\n")
}

func TestSavedSearchAlertService_SendMatchAlerts_PassesSearchCriteriaAsProductFilters(t *testing.T) {
	fixture := newAlertTestFixture(config.NewZerologService())
	subscription := newSubscription("Arabians near Stockholm", "buyer@example.com", "anna", models.SearchCriteria{
		Query: stringPtr("mare"), CategoryID: stringPtr("cat-1"),
		Breed: stringPtr("Arabian"), MaxPrice: floatPtr(50000),
		Lat: floatPtr(59.33), Lng: floatPtr(18.07), RadiusKm: floatPtr(50),
	})
	fixture.expectListingFound()
	fixture.expectSubscriptions(subscription)
	fixture.listings.On("MatchesFilter", mock.Anything, fixture.listingID(), "cat-1", "mare",
		&productmodels.HorseFilter{Breed: stringPtr("Arabian"), MaxPrice: floatPtr(50000)},
		&productmodels.LocationFilter{Lat: 59.33, Lng: 18.07, RadiusKm: 50},
	).Return(false, nil)

	err := fixture.service.SendMatchAlerts(context.Background(), fixture.listingID())

	assert.NoError(t, err)
	fixture.listings.AssertExpectations(t)
}
