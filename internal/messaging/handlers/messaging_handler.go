package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/config"
	"github.com/hfleury/horsemarketplacebk/internal/common"
	"github.com/hfleury/horsemarketplacebk/internal/messaging/models"
	"github.com/hfleury/horsemarketplacebk/internal/messaging/services"
)

type MessagingHandler struct {
	service *services.MessagingService
	logger  config.Logging
}

func NewMessagingHandler(service *services.MessagingService, logger config.Logging) *MessagingHandler {
	return &MessagingHandler{
		service: service,
		logger:  logger,
	}
}

func (h *MessagingHandler) CreateConversation(c *gin.Context) {
	var req models.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewErrorResponse("Invalid payload"))
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}

	conversation, err := h.service.CreateConversation(c.Request.Context(), req, userIDStr.(string))
	if err != nil {
		if err == services.ErrProductNotFound {
			c.JSON(http.StatusNotFound, common.NewErrorResponse(err.Error()))
			return
		}
		if err == services.ErrCannotMessageOwnListing {
			c.JSON(http.StatusForbidden, common.NewErrorResponse(err.Error()))
			return
		}

		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to create conversation", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to create conversation"))
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(conversation))
}

func (h *MessagingHandler) SendMessage(c *gin.Context) {
	conversationID := c.Param("id")

	var req models.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, common.NewErrorResponse("Invalid payload"))
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}

	message, err := h.service.SendMessage(c.Request.Context(), conversationID, req, userIDStr.(string))
	if err != nil {
		if err == services.ErrEmptyMessageBody {
			c.JSON(http.StatusBadRequest, common.NewErrorResponse(err.Error()))
			return
		}
		if err == services.ErrConversationNotFound {
			c.JSON(http.StatusNotFound, common.NewErrorResponse(err.Error()))
			return
		}
		if err == services.ErrNotConversationParticipant {
			c.JSON(http.StatusForbidden, common.NewErrorResponse(err.Error()))
			return
		}

		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to send message", map[string]any{"error": err.Error(), "conversation_id": conversationID})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to send message"))
		return
	}

	c.JSON(http.StatusCreated, common.NewSuccessResponse(message))
}

func (h *MessagingHandler) ListMessages(c *gin.Context) {
	conversationID := c.Param("id")

	afterID, err := strconv.ParseInt(c.Query("after_id"), 10, 64)
	if err != nil {
		afterID = 0
	}

	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}

	messages, hasMore, err := h.service.ListMessages(c.Request.Context(), conversationID, userIDStr.(string), afterID, limit)
	if err != nil {
		if err == services.ErrConversationNotFound {
			c.JSON(http.StatusNotFound, common.NewErrorResponse(err.Error()))
			return
		}
		if err == services.ErrNotConversationParticipant {
			c.JSON(http.StatusForbidden, common.NewErrorResponse(err.Error()))
			return
		}

		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to list messages", map[string]any{"error": err.Error(), "conversation_id": conversationID})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to list messages"))
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(models.ListMessagesResponse{Messages: messages, HasMore: hasMore}))
}

func (h *MessagingHandler) ListConversations(c *gin.Context) {
	page, err := strconv.Atoi(c.Query("page"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}

	result, err := h.service.ListConversations(c.Request.Context(), userIDStr.(string), page, limit)
	if err != nil {
		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to list conversations", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to list conversations"))
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(result))
}

func (h *MessagingHandler) CountUnreadConversations(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewErrorResponse("Unauthorized"))
		return
	}

	count, err := h.service.CountUnreadConversations(c.Request.Context(), userIDStr.(string))
	if err != nil {
		h.logger.Log(c.Request.Context(), config.ErrorLevel, "Failed to count unread conversations", map[string]any{"error": err.Error()})
		c.JSON(http.StatusInternalServerError, common.NewErrorResponse("Failed to count unread conversations"))
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(gin.H{"unread_count": count}))
}
