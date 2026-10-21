package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func runLoggerMiddleware(t *testing.T, rawBody string) (loggedBody string, handlerBody string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	discardLogger := zerolog.New(io.Discard)
	router := gin.New()
	router.Use(LoggerMiddleware(&config.ZerologService{Logger: &discardLogger}))
	router.POST("/login", func(c *gin.Context) {
		loggedBody = c.GetString("request_body")
		bodyBytes, err := io.ReadAll(c.Request.Body)
		assert.NoError(t, err)
		handlerBody = string(bodyBytes)
		c.Status(http.StatusOK)
	})

	request := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(rawBody))
	router.ServeHTTP(httptest.NewRecorder(), request)
	return loggedBody, handlerBody
}

func TestLoggerMiddleware_RedactsLoggedBodyButKeepsRawBodyForHandler(t *testing.T) {
	rawBody := `{"username":"bob","password":"hunter2!"}`

	loggedBody, handlerBody := runLoggerMiddleware(t, rawBody)

	assert.Equal(t, `{"password":"[REDACTED]","username":"bob"}`, loggedBody)
	assert.Equal(t, rawBody, handlerBody)
}

func TestLoggerMiddleware_OmitsUnparseableBody(t *testing.T) {
	rawBody := `{"username":"bob","password":"hun`

	loggedBody, handlerBody := runLoggerMiddleware(t, rawBody)

	assert.Equal(t, fmt.Sprintf("[unparseable body omitted, %d bytes]", len(rawBody)), loggedBody)
	assert.Equal(t, rawBody, handlerBody)
}
