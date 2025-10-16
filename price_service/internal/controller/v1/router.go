package v1

import (
	"net/http"

	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/usamaroman/faas_demo/price_service/docs"

	"github.com/gin-gonic/gin"
	"github.com/usamaroman/faas_demo/price_service/internal/controller/v1/middleware"
	"github.com/usamaroman/faas_demo/price_service/internal/service"
)

func NewRouter(router *gin.Engine, services *service.Services) {
	router.Use(middleware.Log())

	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	ginSwagger.WrapHandler(swaggerfiles.Handler,
		ginSwagger.URL("http://localhost:8085/swagger/doc.json"),
		ginSwagger.DefaultModelsExpandDepth(-1))
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	v1 := router.Group("/v1")
	{
		newTariffRoutes(v1.Group("/tariff"), services.Tariff)
	}
}
