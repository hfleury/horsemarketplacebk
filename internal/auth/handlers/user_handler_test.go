package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/hfleury/horsemarketplacebk/internal/auth/models"
	mockconfig "github.com/hfleury/horsemarketplacebk/internal/mocks/config"
	mockservices "github.com/hfleury/horsemarketplacebk/internal/mocks/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestUserHandler(t *testing.T) (*UserHandler, *mockservices.MockUserServiceInterface) {
	ctrl := gomock.NewController(t)
	mockLogger := mockconfig.NewMockLogging(ctrl)
	mockLogger.EXPECT().GetLoggerFromContext(gomock.Any()).Return(mockLogger).AnyTimes()
	mockLogger.EXPECT().Log(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockService := mockservices.NewMockUserServiceInterface(ctrl)
	return &UserHandler{logger: mockLogger, userService: mockService}, mockService
}

func TestCreateUserHandler_ReturnsSafeUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockService := newTestUserHandler(t)

	userID := uuid.New().String()
	mockService.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Return(&models.UserResponse{
		ID: userID, Username: "new-user", Email: "new@example.com", Role: "user",
	}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := bytes.NewBufferString(`{"username":"new-user","email":"new@example.com","password":"p4ssw0rd!"}`)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/users", body)

	handler.CreateUser(c)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"`+userID+`"`)
	assert.NotContains(t, w.Body.String(), "password")
}

func TestGetAllUsersHandler_ReturnsOnlyAdminFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler, mockService := newTestUserHandler(t)

	userID := uuid.New()
	username, email, role := "listed", "listed@example.com", "user"
	isActive := true
	createdAt := time.Now()
	mockService.EXPECT().GetAllUsers(gomock.Any()).Return([]*models.AdminUserResponse{{
		ID: &userID, Username: &username, Email: &email, Role: &role, IsActive: &isActive, CreatedAt: &createdAt,
	}}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)

	handler.GetAllUsers(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "password")

	var response struct {
		Data []map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	require.Len(t, response.Data, 1)
	keys := make([]string, 0, len(response.Data[0]))
	for key := range response.Data[0] {
		keys = append(keys, key)
	}
	assert.ElementsMatch(t, []string{"id", "username", "email", "role", "is_active", "created_at"}, keys)
}

func TestUserModel_NeverSerialisesPasswordHash(t *testing.T) {
	passwordHash := "$2a$10$hash"
	encoded, err := json.Marshal(models.User{PasswordHash: &passwordHash})

	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "password")
	assert.NotContains(t, string(encoded), passwordHash)
}
