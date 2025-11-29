package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/stretchr/testify/assert"
)

func TestResendHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := config.NewMockLogging(ctrl)
	mockLogger.EXPECT().GetLoggerFromContext(gomock.Any()).Return(mockLogger).AnyTimes()
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockService := services.NewMockUserServiceInterface(ctrl)
	handler := &UserHandler{logger: mockLogger, userService: mockService}

	mockService.EXPECT().ResendVerification(gomock.Any(), "user@example.com").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := bytes.NewBufferString(`{"email":"user@example.com"}`)
	c.Request = httptest.NewRequest("POST", "/api/v1/auth/resend-verification", body)

	handler.ResendVerification(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "If the email exists")
}

func TestResendHandler_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := config.NewMockLogging(ctrl)
	mockLogger.EXPECT().GetLoggerFromContext(gomock.Any()).Return(mockLogger).AnyTimes()
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockService := services.NewMockUserServiceInterface(ctrl)
	handler := &UserHandler{logger: mockLogger, userService: mockService}

	mockService.EXPECT().ResendVerification(gomock.Any(), "user@example.com").Return(assert.AnError)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := bytes.NewBufferString(`{"email":"user@example.com"}`)
	c.Request = httptest.NewRequest("POST", "/api/v1/auth/resend-verification", body)

	handler.ResendVerification(c)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to resend verification")
}
