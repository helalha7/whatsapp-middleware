package webhook

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// token is the verification token that must match the token
// configured in the Meta WhatsApp webhook settings.
const token = "Helal212"

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
