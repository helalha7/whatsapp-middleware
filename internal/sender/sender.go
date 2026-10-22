package sender

import (
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
