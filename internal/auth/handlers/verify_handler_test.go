package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/auth/services"
	"github.com/stretchr/testify/assert"
)

func TestVerifyHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := config.NewMockLogging(ctrl)
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockService := services.NewMockUserService(ctrl)
	handler := &UserHandler{logger: mockLogger, userService: mockService}

	// Expect VerifyEmail to be called with the token
	mockService.EXPECT().VerifyEmail(gomock.Any(), "tok123").Return(nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/auth/verify?token=tok123", nil)
	c.Request.URL.RawQuery = "token=tok123"

	handler.Verify(c)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Email verified")
}

func TestVerifyHandler_Error(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := config.NewMockLogging(ctrl)
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockService := services.NewMockUserService(ctrl)
	handler := &UserHandler{logger: mockLogger, userService: mockService}

	mockService.EXPECT().VerifyEmail(gomock.Any(), "badtoken").Return(assert.AnError)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/auth/verify?token=badtoken", nil)
	c.Request.URL.RawQuery = "token=badtoken"

	handler.Verify(c)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid or expired token")
}
