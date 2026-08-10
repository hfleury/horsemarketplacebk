package geocoding

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hfleury/horsemarketplacebk/internal/common"
)

type GeocodingHandler struct {
	client Client
}

func NewGeocodingHandler(client Client) *GeocodingHandler {
	return &GeocodingHandler{client: client}
}

func (h *GeocodingHandler) Resolve(c *gin.Context) {
	query := c.Query("query")
	if query == "" {
		c.JSON(http.StatusBadRequest, common.NewErrorResponse("query is required"))
		return
	}

	coordinates, err := h.client.Geocode(c.Request.Context(), query)
	if err != nil {
		if errors.Is(err, ErrLocationNotFound) {
			c.JSON(http.StatusNotFound, common.NewErrorResponse("location not found"))
			return
		}
		c.JSON(http.StatusBadGateway, common.NewErrorResponse("failed to resolve location"))
		return
	}

	c.JSON(http.StatusOK, common.NewSuccessResponse(gin.H{"lat": coordinates.Lat, "lng": coordinates.Lng}))
}
