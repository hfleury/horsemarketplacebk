package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct{}

func NewUserHandler() *UserHandler { return &UserHandler{} }

func (h *UserHandler) CreateUser(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "User created"})
}
