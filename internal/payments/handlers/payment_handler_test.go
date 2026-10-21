package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/payments/handlers"
	"github.com/hfleury/horsemarketplacebk/internal/payments/models"
	"github.com/hfleury/horsemarketplacebk/internal/payments/providers"
	"github.com/hfleury/horsemarketplacebk/internal/payments/services"
	"github.com/stretchr/testify/assert"
)

const testUserID = "6f1c2b9e-1111-4c1a-9a51-6f0e8b8f0001"

type fakePaymentService struct {
	providerCodes []string
	checkout      *models.CheckoutResponse
	status        *models.PaymentStatusResponse
	err           error

	lastUserID       string
	lastCheckout     models.CheckoutRequest
	lastPaymentID    string
	lastProviderCode string
	lastHeader       http.Header
	lastBody         []byte
}

func (f *fakePaymentService) ListProviders() []string {
	return f.providerCodes
}

func (f *fakePaymentService) StartCheckout(_ context.Context, userID string, req models.CheckoutRequest) (*models.CheckoutResponse, error) {
	f.lastUserID = userID
	f.lastCheckout = req
	return f.checkout, f.err
}

func (f *fakePaymentService) GetPaymentStatus(_ context.Context, userID, paymentID string) (*models.PaymentStatusResponse, error) {
	f.lastUserID = userID
	f.lastPaymentID = paymentID
	return f.status, f.err
}

func (f *fakePaymentService) HandleProviderEvent(_ context.Context, providerCode string, header http.Header, body []byte) error {
	f.lastProviderCode = providerCode
	f.lastHeader = header
	f.lastBody = body
	return f.err
}

// newRouter mirrors internal/router/payment_router.go. A non-empty userID
// stands in for RequireAuth having set user_id.
func newRouter(service *fakePaymentService, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	handler := handlers.NewPaymentHandler(service, config.NewZerologService())
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if userID != "" {
			c.Set("user_id", userID)
		}
	})
	router.GET("/payments/providers", handler.ListProviders)
	router.POST("/payments/webhooks/:provider", handler.Webhook)
	router.POST("/payments/checkout", handler.Checkout)
	router.GET("/payments/:id", handler.GetStatus)
	return router
}

func serve(router *gin.Engine, method, path, body string, header http.Header) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	for key, values := range header {
		request.Header[key] = values
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

const validCheckoutBody = `{"product_id":"p1","tier_code":"medium","provider":"sandbox","starts_at":"2026-12-24T09:00:00+01:00"}`

func TestPaymentHandler_ListProviders(t *testing.T) {
	tests := []struct {
		name  string
		codes []string
		want  string
	}{
		{name: "registered providers", codes: []string{"sandbox"}, want: `"data":["sandbox"]`},
		{name: "none registered is an empty array", codes: []string{}, want: `"data":[]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := serve(newRouter(&fakePaymentService{providerCodes: tt.codes}, ""), http.MethodGet, "/payments/providers", "", nil)

			assert.Equal(t, http.StatusOK, recorder.Code)
			assert.Contains(t, recorder.Body.String(), tt.want)
		})
	}
}

func TestPaymentHandler_Checkout_Success(t *testing.T) {
	paymentID := uuid.New()
	service := &fakePaymentService{checkout: &models.CheckoutResponse{PaymentID: paymentID, CheckoutURL: "https://pay.example"}}

	recorder := serve(newRouter(service, testUserID), http.MethodPost, "/payments/checkout", validCheckoutBody, nil)

	assert.Equal(t, http.StatusCreated, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"checkout_url":"https://pay.example"`)
	assert.Contains(t, recorder.Body.String(), paymentID.String())
	assert.Equal(t, testUserID, service.lastUserID)
	assert.Equal(t, "p1", service.lastCheckout.ProductID)
	assert.Equal(t, "medium", service.lastCheckout.TierCode)
	assert.Equal(t, "sandbox", service.lastCheckout.Provider)
	assert.Equal(t, "2026-12-24T08:00:00Z", service.lastCheckout.StartsAt.UTC().Format("2006-01-02T15:04:05Z07:00"))
}

func TestPaymentHandler_Checkout_RequestRejected(t *testing.T) {
	tests := []struct {
		name   string
		userID string
		body   string
		want   int
	}{
		{name: "no user", body: validCheckoutBody, want: http.StatusUnauthorized},
		{name: "malformed JSON", userID: testUserID, body: `{`, want: http.StatusBadRequest},
		{name: "missing tier", userID: testUserID, body: `{"product_id":"p1","provider":"sandbox"}`, want: http.StatusBadRequest},
		{name: "missing provider", userID: testUserID, body: `{"product_id":"p1","tier_code":"medium"}`, want: http.StatusBadRequest},
		{name: "start date not RFC 3339", userID: testUserID, body: `{"product_id":"p1","tier_code":"medium","provider":"sandbox","starts_at":"tomorrow"}`, want: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakePaymentService{}

			recorder := serve(newRouter(service, tt.userID), http.MethodPost, "/payments/checkout", tt.body, nil)

			assert.Equal(t, tt.want, recorder.Code)
			assert.Empty(t, service.lastUserID, "the service must not be called")
		})
	}
}

func TestPaymentHandler_ServiceErrorMapping(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{err: fmt.Errorf("%w: product id must be a UUID", services.ErrInvalidPaymentRequest), want: http.StatusBadRequest},
		{err: fmt.Errorf("%w: %q", services.ErrUnknownProvider, "stripe"), want: http.StatusBadRequest},
		{err: fmt.Errorf("%w: %q", services.ErrTierNotFound, "gold"), want: http.StatusBadRequest},
		{err: services.ErrNotListingOwner, want: http.StatusForbidden},
		{err: services.ErrListingNotFound, want: http.StatusNotFound},
		{err: services.ErrPaymentNotFound, want: http.StatusNotFound},
		{err: services.ErrListingNotBoostable, want: http.StatusConflict},
		{err: fmt.Errorf("%w: timeout", services.ErrProviderUnavailable), want: http.StatusBadGateway},
		{err: errors.New("db down"), want: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			for _, request := range []struct{ method, path, body string }{
				{http.MethodPost, "/payments/checkout", validCheckoutBody},
				{http.MethodGet, "/payments/" + uuid.NewString(), ""},
			} {
				recorder := serve(newRouter(&fakePaymentService{err: tt.err}, testUserID), request.method, request.path, request.body, nil)
				assert.Equal(t, tt.want, recorder.Code, request.path)
			}
		})
	}
}

func TestPaymentHandler_ProviderUnavailableHidesProviderDetail(t *testing.T) {
	service := &fakePaymentService{err: fmt.Errorf("%w: secret key sk_live_123 rejected", services.ErrProviderUnavailable)}

	recorder := serve(newRouter(service, testUserID), http.MethodPost, "/payments/checkout", validCheckoutBody, nil)

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "sk_live_123")
}

func TestPaymentHandler_GetStatus(t *testing.T) {
	paymentID := uuid.New()
	service := &fakePaymentService{status: &models.PaymentStatusResponse{PaymentID: paymentID, Status: models.StatusPaid, Tier: "pro"}}

	recorder := serve(newRouter(service, testUserID), http.MethodGet, "/payments/"+paymentID.String(), "", nil)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"status":"paid"`)
	assert.Equal(t, paymentID.String(), service.lastPaymentID)
	assert.Equal(t, testUserID, service.lastUserID)
}

func TestPaymentHandler_GetStatus_NoUser(t *testing.T) {
	recorder := serve(newRouter(&fakePaymentService{}, ""), http.MethodGet, "/payments/"+uuid.NewString(), "", nil)

	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestPaymentHandler_ProvidersRouteIsNotAPaymentID(t *testing.T) {
	service := &fakePaymentService{providerCodes: []string{"sandbox"}}

	recorder := serve(newRouter(service, testUserID), http.MethodGet, "/payments/providers", "", nil)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"data":["sandbox"]`)
	assert.Empty(t, service.lastPaymentID, "GetStatus must not handle /payments/providers")
}

func TestPaymentHandler_Webhook(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{name: "applied, ignored or duplicate", want: http.StatusOK},
		{name: "bad signature", err: providers.ErrInvalidSignature, want: http.StatusBadRequest},
		{name: "malformed event", err: fmt.Errorf("%w: reference is required", providers.ErrMalformedEvent), want: http.StatusBadRequest},
		{name: "unknown provider", err: fmt.Errorf("%w: %q", services.ErrUnknownProvider, "stripe"), want: http.StatusNotFound},
		{name: "unknown payment is retried", err: services.ErrPaymentNotFound, want: http.StatusNotFound},
		{name: "unexpected failure is retried", err: errors.New("db down"), want: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakePaymentService{err: tt.err}
			header := http.Header{"X-Sandbox-Secret": {"s3cret"}}

			recorder := serve(newRouter(service, ""), http.MethodPost, "/payments/webhooks/sandbox", `{"reference":"sbx_1"}`, header)

			assert.Equal(t, tt.want, recorder.Code)
			assert.Equal(t, "sandbox", service.lastProviderCode)
			assert.Equal(t, "s3cret", service.lastHeader.Get("X-Sandbox-Secret"))
			assert.Equal(t, `{"reference":"sbx_1"}`, string(service.lastBody))
		})
	}
}

func TestPaymentHandler_Webhook_BodySizeLimit(t *testing.T) {
	tests := []struct {
		name string
		size int
		want int
	}{
		{name: "exactly 64 KiB", size: 64 << 10, want: http.StatusOK},
		{name: "over 64 KiB", size: 64<<10 + 1, want: http.StatusRequestEntityTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakePaymentService{}

			recorder := serve(newRouter(service, ""), http.MethodPost, "/payments/webhooks/sandbox", strings.Repeat("a", tt.size), nil)

			assert.Equal(t, tt.want, recorder.Code)
			assert.Equal(t, tt.want == http.StatusOK, service.lastProviderCode != "", "an oversize body must not reach the service")
		})
	}
}
