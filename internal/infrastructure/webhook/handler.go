package webhook

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// token is the verification token that must match the token
// configured in the Meta WhatsApp webhook settings.
const token = "Helal212"
const accessToken = "EAALq29NMSvMBRPeGiZBQ7iZA3y3OyeqnPIfErWDI0eQf1UobLcWpoZCyFIRt2etU4u24bWCZBAFAR9MaCvZCIwcYkQ6GbNEeKFR1ORz642AtRXDHBtgmIWIn1ZBckDKx3GQoLAPijRWUVx5NhsRLVvAZAiANfT7sJMrZByfmRqJJeV7CFBlLVyEeZBA8jPg5QWHmmZAhMIxrnyZCk0o5P5HhTixILXIMA1ZCuLt3OPAhSWQSAvHicZAiiuD25iXU5ww44fgI2mLEL0HlRaHnafMnWL006A739"

type Handler struct {
}

// VerifyWebhook handles the webhook verification challenge from Meta.
func (handler *Handler) VerifyWebhook(c *gin.Context) {
	challenge := c.Query("hub.challenge")
	tok := c.Query("hub.verify_token")
	mode := c.Query("hub.mode")

	if tok == token && mode == "subscribe" {
		c.String(http.StatusOK, challenge)
		return
	}
	c.Status(http.StatusBadRequest)
}

// HandleWebhook handles incoming WhatsApp webhook events.
func (handler *Handler) HandleWebhook(c *gin.Context) {
	payload := &WebhookPayload{}

	err := c.BindJSON(payload)
	if err != nil {
		c.Status(http.StatusBadRequest)
	}

	fmt.Println(payload)

	c.Status(http.StatusOK)
}
