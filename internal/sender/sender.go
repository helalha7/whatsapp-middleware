package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Sender struct {
	client *http.Client
}

func New(
	client *http.Client,
) *Sender {

	return &Sender{
		client: client,
	}
}

func (s *Sender) SendTextMessage(
	senderPhoneNumberID string,
	senderAccessToken string,
	recieverPhoneNumber string,
	message string,
) error {
	body := struct {
		MessagingProduct string `json:"messaging_product"`
		RecipientType    string `json:"recipient_type"`
		To               string `json:"to"`
		MessageType      string `json:"type"`
		Text             struct {
			PreviewURL bool   `json:"preview_url"`
			Body       string `json:"body"`
		} `json:"text"`
	}{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               recieverPhoneNumber,
		MessageType:      "text",
	}

	body.Text.PreviewURL = false
	body.Text.Body = message

	js, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"https://graph.facebook.com/v26.0/%s/messages",
		senderPhoneNumberID,
	)
	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		url,
		bytes.NewReader(js),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+senderAccessToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}
