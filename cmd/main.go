package main

import (
	"log"
	"net/http"
	"os"

	"github.com/helalha7/whatsapp-middleware/internal/listener"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	listener.Register(mux, os.Getenv("verify_token"), os.Getenv("app_secret"))

	if err := Serve(mux); err != nil {
		log.Fatal(err)
	}
}
