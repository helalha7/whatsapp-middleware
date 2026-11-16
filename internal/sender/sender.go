package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultBaseURL = "https://graph.facebook.com/v26.0"

type Sender struct {
	client  *http.Client
	baseURL string
}

func New(
	client *http.Client,
) *Sender {

	return &Sender{
		client:  client,
		baseURL: defaultBaseURL,
	}
}

func (s *Sender) SendTextMessage(
	ctx context.Context,
	senderPhoneNumberID string,
	senderAccessToken string,
	receiverPhoneNumber string,
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
		To:               receiverPhoneNumber,
		MessageType:      "text",
	}

	body.Text.PreviewURL = false
	body.Text.Body = message

	js, err := json.Marshal(body)
	if err != nil {
		return err
	}

	url := fmt.Sprintf(
		"%s/%s/messages",
		strings.TrimRight(s.baseURL, "/"),
		senderPhoneNumberID,
	)
	req, err := http.NewRequestWithContext(
		ctx,
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
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		if readErr != nil {
			return readErr
		}

		return fmt.Errorf(
			"WhatsApp API returned %s: %s",
			resp.Status,
			strings.TrimSpace(string(responseBody)),
		)
	}

	return nil
}
