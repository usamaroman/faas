package v1

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/usamaroman/faas_demo/price_service/internal/controller/v1/request"
	"github.com/usamaroman/faas_demo/price_service/internal/controller/v1/response"
	_ "github.com/usamaroman/faas_demo/price_service/internal/entity"
	"github.com/usamaroman/faas_demo/price_service/internal/service"
)

type tariffRoutes struct {
	valid *validator.Validate

	tariffService service.Tariff
}

func newTariffRoutes(g *gin.RouterGroup, tariffService service.Tariff) {
	slog.Debug("component", "tariff routes")

	v := validator.New()

	r := &tariffRoutes{
		valid:         v,
		tariffService: tariffService,
	}

	g.POST("/", r.createNewTariff)
	g.GET("/:id", r.getTariffByID)
	g.GET("/", r.getTariffs)
	g.PATCH("/:id", r.updateTariffByID)
	g.DELETE("/:id", r.deleteTariffByID)
}

// @Summary Создание нового тарифа
// @Description Создание нового тарифа
// @Tags тарифы
// @Accept json
// @Param input body request.CreateTariff true "Тело запроса"
// @Success 201 {object} entity.Tariff
// @Router /v1/tariff/ [post]
func (r *tariffRoutes) createNewTariff(c *gin.Context) {
	var tariff request.CreateTariff

	err := c.ShouldBindJSON(&tariff)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})

		return
	}

	err = r.valid.Struct(&tariff)
	if err != nil {
		slog.Info("error validating data", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})

		return
	}

	createdTariff, err := r.tariffService.Create(c, &service.TariffInput{
		Name:      tariff.Name,
		ExecPrice: tariff.ExecPrice,
		MemPrice:  tariff.MemPrice,
		CpuPrice:  tariff.CpuPrice,
	})
	if err != nil {
		slog.Error("failed to create tariff", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	slog.Info("created new tariff")
	c.JSON(http.StatusCreated, gin.H{
		"tariff": createdTariff,
	})
}

// @Summary Получить тариф по идентификатору
// @Description Получить тариф по идентификатору
// @Tags тарифы
// @Produce json
// @Param id path int true "Идентификатор тарифа"
// @Success 200 {object} entity.Tariff
// @Router /v1/tariff/{id} [get]
func (r *tariffRoutes) getTariffByID(c *gin.Context) {
	tariffID := c.Param("id")

	id, err := strconv.Atoi(tariffID)
	if err != nil {
		slog.Error("invalid id parameter", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id parameter",
		})
		return
	}

	tariff, err := r.tariffService.GetByID(c, id)
	if err != nil {
		if errors.Is(err, service.ErrTariffNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": service.ErrTariffNotFound.Error(),
			})
			return
		}

		slog.Error("failed to get tariff", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tariff": tariff,
	})
}

// @Summary Получить все тарифы
// @Description Получить все тарифы
// @Tags тарифы
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} response.GetAllTariffs
// @Router /v1/tariff [get]
func (r *tariffRoutes) getTariffs(c *gin.Context) {
	filters := buildTariffFilters(c)
	if filters == nil {
		return
	}

	tariffs, err := r.tariffService.GetAll(c, filters)
	if err != nil {
		slog.Error("failed to get all tariffs", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, response.GetAllTariffs{
		Tariffs: tariffs,
	})
}

// @Summary Обновить тариф по его идентификатору
// @Description Обновить тариф по его идентификатору. Принимает JSON с обновленными полями
// @Tags тарифы
// @Accept json
// @Param id path int true "Идентификатор тарифа"
// @Param input body request.UpdateTariff true "Тело запроса"
// @Success 200 {object} entity.Tariff
// @Router /v1/tariff/{id} [patch]
func (r *tariffRoutes) updateTariffByID(c *gin.Context) {
	tariffID := c.Param("id")

	id, err := strconv.Atoi(tariffID)
	if err != nil {
		slog.Error("invalid id parameter", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id parameter",
		})
		return
	}

	var updateData request.UpdateTariff

	if err := c.ShouldBindJSON(&updateData); err != nil {
		slog.Error("Invalid JSON payload", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON payload",
		})
		return
	}

	updatedTariff, err := r.tariffService.UpdateByID(c, id, &service.TariffInput{
		Name:      updateData.Name,
		ExecPrice: updateData.ExecPrice,
		MemPrice:  updateData.MemPrice,
		CpuPrice:  updateData.CpuPrice,
	})
	if err != nil {
		if errors.Is(err, service.ErrTariffNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": service.ErrTariffNotFound.Error(),
			})
			return
		}

		slog.Error("failed to update tariff", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	slog.Info("successfully updated tariff", slog.String("tariff id", tariffID))
	c.JSON(http.StatusOK, gin.H{
		"tariff": updatedTariff,
	})
}

// @Summary Удалить тариф по его идентификатору
// @Description Удалить тариф по его идентификатору
// @Tags тарифы
// @Param id path int true "Идентификатор тарифа"
// @Success 204 "No Content"
// @Router /v1/tariff/{id} [delete]
func (r *tariffRoutes) deleteTariffByID(c *gin.Context) {
	tariffID := c.Param("id")

	id, err := strconv.Atoi(tariffID)
	if err != nil {
		slog.Error("invalid id parameter", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id parameter",
		})
		return
	}

	err = r.tariffService.DeleteByID(c, id)
	if err != nil {
		if errors.Is(err, service.ErrTariffNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": service.ErrTariffNotFound.Error(),
			})
			return
		}

		slog.Error("failed to delete tariff", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	slog.Info("deleted tariff", slog.String("tariffID", tariffID))
	c.Status(http.StatusNoContent)
}
