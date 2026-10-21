package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/common"
	"github.com/hfleury/horsemarketplacebk/internal/payments/models"
	"github.com/hfleury/horsemarketplacebk/internal/payments/providers"
	"github.com/hfleury/horsemarketplacebk/internal/payments/services"
)

// maxWebhookBodyBytes is far above any provider event we expect.
const maxWebhookBodyBytes = 64 << 10

// paymentService is what the handler needs from services.PaymentService.
type paymentService interface {
	ListProviders() []string
	StartCheckout(ctx context.Context, userID string, req models.CheckoutRequest) (*models.CheckoutResponse, error)
	GetPaymentStatus(ctx context.Context, userID, paymentID string) (*models.PaymentStatusResponse, error)
	HandleProviderEvent(ctx context.Context, providerCode string, header http.Header, body []byte) error
}

type PaymentHandler struct {
	service paymentService
	logger  config.Logging
}

func NewPaymentHandler(service paymentService, logger config.Logging) *PaymentHandler {
	return &PaymentHandler{
		service: service,
		logger:  logger,
	}
}

func (h *PaymentHandler) ListProviders(c *gin.Context) {
	c.JSON(http.StatusOK, common.NewSuccessResponse(h.service.ListProviders()))
}

func (h *PaymentHandler) Checkout(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}
	var req models.CheckoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewErrorResponse("Invalid payload"))
		return
	}

	checkout, err := h.service.StartCheckout(c.Request.Context(), userID, req)
	if err != nil {
		h.respondWithError(c, err, "Failed to start checkout")
		return
	}

	c.JSON(http.StatusCreated, common.NewSuccessResponse(checkout))
}

func (h *PaymentHandler) GetStatus(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	status, err := h.service.GetPaymentStatus(c.Request.Context(), userID, c.Param("id"))
	if err != nil {
		h.respondWithError(c, err, "Failed to get payment status")
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(status))
}

// Webhook receives provider callbacks. It never logs the body, which may hold
// customer details. Any non-2xx reply makes the provider retry the event.
func (h *PaymentHandler) Webhook(c *gin.Context) {
	providerCode := c.Param("provider")
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookBodyBytes))
	if err != nil {
		h.rejectWebhook(c, http.StatusRequestEntityTooLarge, providerCode, err)
		return
	}

	err = h.service.HandleProviderEvent(c.Request.Context(), providerCode, c.Request.Header, body)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, common.NewSuccessResponse(nil))
	case errors.Is(err, providers.ErrInvalidSignature), errors.Is(err, providers.ErrMalformedEvent):
		h.rejectWebhook(c, http.StatusBadRequest, providerCode, err)
	case errors.Is(err, services.ErrUnknownProvider), errors.Is(err, services.ErrPaymentNotFound):
		h.rejectWebhook(c, http.StatusNotFound, providerCode, err)
	default:
		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to handle payment provider event", map[string]any{"error": err.Error(), "provider": providerCode})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to handle payment event"))
	}
}

// rejectWebhook logs at Warn: these are expected client-side failures and
// must not open Sentry issues.
func (h *PaymentHandler) rejectWebhook(c *gin.Context, status int, providerCode string, err error) {
	h.logger.Log(c.Request.Context(), config.WarnLevel, "Rejected payment provider event", map[string]any{"error": err.Error(), "provider": providerCode, "status": status})
	c.JSON(status, common.NewErrorResponse(err.Error()))
}

func requireUserID(c *gin.Context) (string, bool) {
	value, exists := c.Get("user_id")
	userID, isString := value.(string)
	if !exists || !isString {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return "", false
	}
	return userID, true
}

// respondWithError uses errors.Is because request errors are wrapped with
// the specific rule that failed.
func (h *PaymentHandler) respondWithError(c *gin.Context, err error, logMessage string) {
	switch {
	case errors.Is(err, services.ErrInvalidPaymentRequest),
		errors.Is(err, services.ErrUnknownProvider),
		errors.Is(err, services.ErrTierNotFound):
		c.JSON(http.StatusBadRequest, common.NewErrorResponse(err.Error()))
	case errors.Is(err, services.ErrNotListingOwner):
		c.JSON(http.StatusForbidden, common.NewErrorResponse(err.Error()))
	case errors.Is(err, services.ErrListingNotFound),
		errors.Is(err, services.ErrPaymentNotFound):
		c.JSON(http.StatusNotFound, common.NewErrorResponse(err.Error()))
	case errors.Is(err, services.ErrListingNotBoostable):
		c.JSON(http.StatusConflict, common.NewErrorResponse(err.Error()))
	case errors.Is(err, services.ErrProviderUnavailable):
		c.JSON(http.StatusBadGateway, common.NewErrorResponse(services.ErrProviderUnavailable.Error()))
	default:
		h.logger.Log(c.Request.Context(), config.ErrorLevel, logMessage, map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse(logMessage))
	}
}
