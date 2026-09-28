package services_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	authModels "github.com/hfleury/horsemarketplacebk/internal/auth/models"
	"github.com/hfleury/horsemarketplacebk/internal/callbacks/models"
	"github.com/hfleury/horsemarketplacebk/internal/callbacks/services"
	mockuserrepo "github.com/hfleury/horsemarketplacebk/internal/mocks/auth/repositories"
	mockcallbacks "github.com/hfleury/horsemarketplacebk/internal/mocks/callbacks"
	mockemail "github.com/hfleury/horsemarketplacebk/internal/mocks/email"
	mockProducts "github.com/hfleury/horsemarketplacebk/internal/mocks/products"
	productModels "github.com/hfleury/horsemarketplacebk/internal/products/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSubmit_Success(t *testing.T) {
	mockCallbackRepo := new(mockcallbacks.MockCallbackRequestRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()

	ctrl := gomock.NewController(t)
	mockUserRepo := mockuserrepo.NewMockUserRepository(ctrl)
	sender := mockemail.NewMockSender()

	service := services.NewCallbackRequestService(mockCallbackRepo, mockProductRepo, mockUserRepo, logger)
	service.SetEmailSender(sender)

	productID := uuid.New()
	sellerID := uuid.New()
	buyerID := uuid.New()
	sellerEmail := "seller@example.com"

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(&productModels.Product{ID: productID, UserID: sellerID}, nil)
	mockCallbackRepo.On("Create", mock.Anything, mock.MatchedBy(func(r *models.CallbackRequest) bool {
		return r.ProductID == productID && r.BuyerID == buyerID && r.PhoneNumber == "+46 70 123 45 67"
	})).Return(&models.CallbackRequest{ID: uuid.New(), ProductID: productID, BuyerID: buyerID, PhoneNumber: "+46 70 123 45 67"}, nil)
	mockUserRepo.EXPECT().SelectUserByID(gomock.Any(), sellerID.String()).Return(&authModels.User{Id: &sellerID, Email: &sellerEmail}, nil)

	req := models.CreateCallbackRequestRequest{ProductID: productID.String(), PhoneNumber: "+46 70 123 45 67"}

	created, err := service.Submit(context.Background(), req, buyerID.String())

	assert.NoError(t, err)
	assert.Equal(t, buyerID, created.BuyerID)
	assert.Equal(t, sellerEmail, sender.LastTo)
	assert.Equal(t, "Callback request on HorseMarketplace", sender.LastSubject)
	assert.Contains(t, sender.LastBody, "+46 70 123 45 67")
	mockCallbackRepo.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
}

func TestSubmit_InvalidPhoneNumber(t *testing.T) {
	mockCallbackRepo := new(mockcallbacks.MockCallbackRequestRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewCallbackRequestService(mockCallbackRepo, mockProductRepo, nil, logger)

	req := models.CreateCallbackRequestRequest{ProductID: uuid.New().String(), PhoneNumber: "abc"}

	_, err := service.Submit(context.Background(), req, uuid.New().String())

	assert.ErrorIs(t, err, services.ErrInvalidPhoneNumber)
	mockProductRepo.AssertNotCalled(t, "FindByID")
	mockCallbackRepo.AssertNotCalled(t, "Create")
}

func TestSubmit_ProductNotFound(t *testing.T) {
	mockCallbackRepo := new(mockcallbacks.MockCallbackRequestRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewCallbackRequestService(mockCallbackRepo, mockProductRepo, nil, logger)

	productID := uuid.New()
	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(nil, nil)

	req := models.CreateCallbackRequestRequest{ProductID: productID.String(), PhoneNumber: "0701234567"}

	_, err := service.Submit(context.Background(), req, uuid.New().String())

	assert.ErrorIs(t, err, services.ErrProductNotFound)
	mockCallbackRepo.AssertNotCalled(t, "Create")
}

func TestSubmit_CannotRequestOwnListing(t *testing.T) {
	mockCallbackRepo := new(mockcallbacks.MockCallbackRequestRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewCallbackRequestService(mockCallbackRepo, mockProductRepo, nil, logger)

	productID := uuid.New()
	ownerID := uuid.New()

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(&productModels.Product{ID: productID, UserID: ownerID}, nil)

	req := models.CreateCallbackRequestRequest{ProductID: productID.String(), PhoneNumber: "0701234567"}

	_, err := service.Submit(context.Background(), req, ownerID.String())

	assert.ErrorIs(t, err, services.ErrCannotRequestOwnListing)
	mockCallbackRepo.AssertNotCalled(t, "Create")
}

func TestSubmit_SuccessWithNilEmailSender(t *testing.T) {
	mockCallbackRepo := new(mockcallbacks.MockCallbackRequestRepository)
	mockProductRepo := new(mockProducts.MockProductRepo)
	logger := config.NewZerologService()
	service := services.NewCallbackRequestService(mockCallbackRepo, mockProductRepo, nil, logger)

	productID := uuid.New()
	sellerID := uuid.New()
	buyerID := uuid.New()

	mockProductRepo.On("FindByID", mock.Anything, productID.String()).Return(&productModels.Product{ID: productID, UserID: sellerID}, nil)
	mockCallbackRepo.On("Create", mock.Anything, mock.Anything).Return(&models.CallbackRequest{ID: uuid.New(), ProductID: productID, BuyerID: buyerID, PhoneNumber: "0701234567"}, nil)

	req := models.CreateCallbackRequestRequest{ProductID: productID.String(), PhoneNumber: "0701234567"}

	created, err := service.Submit(context.Background(), req, buyerID.String())

	assert.NoError(t, err)
	assert.Equal(t, buyerID, created.BuyerID)
	mockCallbackRepo.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
}
