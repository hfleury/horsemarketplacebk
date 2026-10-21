package services

import (
	"context"
	"fmt"
	"time"
	_ "time/tzdata" // the production image has no zoneinfo, and the email shows Swedish time

	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/analytics"
	"github.com/hfleury/horsemarketplacebk/internal/common"
	"github.com/hfleury/horsemarketplacebk/internal/payments/models"
)

var swedishTime = mustLoadLocation("Europe/Stockholm")

func mustLoadLocation(name string) *time.Location {
	location, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return location
}

func (s *PaymentService) trackBoostPaid(ctx context.Context, providerCode string, confirmed *models.ConfirmedPayment) {
	if confirmed.UserID == nil {
		return
	}
	s.tracker.Track(ctx, analytics.Event{
		Name:       analytics.EventBoostPaid,
		DistinctID: confirmed.UserID.String(),
		Properties: map[string]any{
			"listing_id": confirmed.Boost.ProductID.String(),
			"tier":       confirmed.Boost.Tier,
			"amount_sek": confirmed.AmountSEK,
			"provider":   providerCode,
		},
	})
}

// notifyBoostConfirmed only logs its failures: the payment is already
// committed, and an error here would make the provider retry the event.
func (s *PaymentService) notifyBoostConfirmed(ctx context.Context, confirmed *models.ConfirmedPayment) {
	if s.emailSender == nil || confirmed.UserID == nil {
		return
	}
	paymentID := confirmed.PaymentID.String()

	seller, err := s.users.SelectUserByID(ctx, confirmed.UserID.String())
	if err != nil {
		s.logger.Log(ctx, config.ErrorLevel, "Failed to look up seller for boost confirmation", map[string]any{"error": err.Error(), "paymentId": paymentID})
		return
	}
	if seller == nil || seller.Email == nil {
		return
	}
	listing, err := s.listings.FindByID(ctx, confirmed.Boost.ProductID.String())
	if err != nil {
		s.logger.Log(ctx, config.ErrorLevel, "Failed to look up listing for boost confirmation", map[string]any{"error": err.Error(), "paymentId": paymentID})
		return
	}
	if listing == nil {
		return
	}

	subject, body := buildBoostConfirmationEmail(listing.Title, confirmed, s.frontendURL)
	if err := s.emailSender.Send(ctx, *seller.Email, subject, body); err != nil {
		s.logger.Log(ctx, config.ErrorLevel, "Failed to send boost confirmation email", map[string]any{"error": err.Error(), "paymentId": paymentID})
	}
}

func buildBoostConfirmationEmail(listingTitle string, confirmed *models.ConfirmedPayment, frontendURL string) (subject, body string) {
	boost := confirmed.Boost
	subject = fmt.Sprintf("Your boost for %q is confirmed", listingTitle)
	body = fmt.Sprintf(
		"Hi,\n\nWe have received your payment of %s (incl. moms) for the %q boost on %q.\n\nYour boost runs from %s to %s (Swedish time).\n\nView the listing: %s/listings/%s",
		common.FormatPriceSEK(&confirmed.AmountSEK), boost.Tier, listingTitle,
		formatSwedishTime(boost.StartsAt), formatSwedishTime(boost.ExpiresAt),
		frontendURL, boost.ProductID,
	)
	return subject, body
}

func formatSwedishTime(moment time.Time) string {
	return moment.In(swedishTime).Format("2006-01-02 15:04")
}
