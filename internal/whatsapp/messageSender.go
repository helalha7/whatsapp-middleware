package whatsapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type SendMessageRequest struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             Text   `json:"text"`
}

func SendTextMessage(accessToken, phoneNumberID, to, message string) error {
	url := fmt.Sprintf("https://graph.facebook.com/v25.0/%s/messages", phoneNumberID)

	reqBody := SendMessageRequest{
		MessagingProduct: "whatsapp",
		To:               to,
		Type:             "text",
		Text: Text{
			Body: message,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed to send message, status: %s", resp.Status)
	}

	return nil
}
