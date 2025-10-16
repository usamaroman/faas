package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/usamaroman/faas_demo/price_service/internal/entity"
)

func buildTariffFilters(c *gin.Context) *entity.TariffFilters {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.ParseUint(limitStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid limit parameter",
		})
		return nil
	}

	offset, err := strconv.ParseUint(offsetStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid offset parameter",
		})
		return nil
	}

	return &entity.TariffFilters{
		Limit:  limit,
		Offset: offset,
	}
}
