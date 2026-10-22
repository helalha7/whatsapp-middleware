package main

import (
	"log"
	"net/http"
	"os"

	"github.com/helalha7/whatsapp-middleware/internal/listener"
	"github.com/joho/godotenv"
)

type application struct {
	listener *listener.Handler
}

func main() {
	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal(err)
	}

	app := &application{
		listener: &listener.Handler{
			VerifyToken: os.Getenv("verify_token"),
			AppSecret:   os.Getenv("app_secret"),
		},
	}

	err = http.ListenAndServe(":8080", app.Routes())
	if err != nil {
		log.Fatal(err)
	}
}
