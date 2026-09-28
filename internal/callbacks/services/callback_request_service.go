package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/config"
	authRepositories "github.com/hfleury/horsemarketplacebk/internal/auth/repositories"
	"github.com/hfleury/horsemarketplacebk/internal/callbacks/models"
	"github.com/hfleury/horsemarketplacebk/internal/callbacks/repositories"
	"github.com/hfleury/horsemarketplacebk/internal/email"
	productRepositories "github.com/hfleury/horsemarketplacebk/internal/products/repositories"
)

var (
	ErrProductNotFound         = errors.New("product not found")
	ErrCannotRequestOwnListing = errors.New("cannot request a callback on your own listing")
	ErrInvalidPhoneNumber      = errors.New("phone number must be 6-20 characters and contain only digits, spaces, and + - ( )")
)

var phoneNumberPattern = regexp.MustCompile(`^[0-9+\-\s()]{6,20}$`)

type CallbackRequestService struct {
	repo        repositories.CallbackRequestRepository
	productRepo productRepositories.ProductRepository
	userRepo    authRepositories.UserRepository
	emailSender email.Sender
	logger      config.Logging
}

func NewCallbackRequestService(repo repositories.CallbackRequestRepository, productRepo productRepositories.ProductRepository, userRepo authRepositories.UserRepository, logger config.Logging) *CallbackRequestService {
	return &CallbackRequestService{
		repo:        repo,
		productRepo: productRepo,
		userRepo:    userRepo,
		logger:      logger,
	}
}

// SetEmailSender allows wiring an email.Sender after construction without
// changing existing constructor call sites.
func (s *CallbackRequestService) SetEmailSender(sender email.Sender) {
	s.emailSender = sender
}

func (s *CallbackRequestService) Submit(ctx context.Context, req models.CreateCallbackRequestRequest, buyerUserID string) (*models.CallbackRequest, error) {
	if !phoneNumberPattern.MatchString(req.PhoneNumber) {
		return nil, ErrInvalidPhoneNumber
	}

	product, err := s.productRepo.FindByID(ctx, req.ProductID)
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}

	if product.UserID.String() == buyerUserID {
		return nil, ErrCannotRequestOwnListing
	}

	buyerUUID, err := uuid.Parse(buyerUserID)
	if err != nil {
		return nil, err
	}

	callbackRequest := &models.CallbackRequest{
		ProductID:   product.ID,
		BuyerID:     buyerUUID,
		PhoneNumber: req.PhoneNumber,
	}

	created, err := s.repo.Create(ctx, callbackRequest)
	if err != nil {
		return nil, err
	}

	s.notifySeller(ctx, product.UserID, created)

	return created, nil
}

func (s *CallbackRequestService) notifySeller(ctx context.Context, sellerID uuid.UUID, request *models.CallbackRequest) {
	if s.emailSender == nil {
		return
	}

	seller, err := s.userRepo.SelectUserByID(ctx, sellerID.String())
	if err != nil {
		s.logger.Log(ctx, config.ErrorLevel, "failed to look up callback request notification recipient", map[string]any{"error": err.Error()})
		return
	}
	if seller == nil || seller.Email == nil {
		return
	}

	body := fmt.Sprintf("A buyer has requested a callback on your HorseMarketplace listing.\n\nCall them back at: %s", request.PhoneNumber)
	if err := s.emailSender.Send(ctx, *seller.Email, "Callback request on HorseMarketplace", body); err != nil {
		s.logger.Log(ctx, config.ErrorLevel, "failed to send callback request notification email", map[string]any{"error": err.Error()})
	}
}
