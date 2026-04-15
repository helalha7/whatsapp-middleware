package webhook

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(router *gin.Engine, h *Handler) {
	router.GET("/webhook", h.VerifyWebhook)
	router.POST("/webhook", h.HandleWebhook)
}
