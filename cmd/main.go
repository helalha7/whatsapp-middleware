package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/helalha7/whatsapp-middleware/internal/listener"
	"github.com/joho/godotenv"
)

func main() {
	//getting the environment type, maybe we will need it more for later
	env := flag.String("env", "development", "set the environment prudction or development")
	flag.Parse()

	//if we are in development we need to load the environment variables
	if *env == "development" {
		err := godotenv.Load("./.env")
		if err != nil {
			log.Fatal(err)
		}
	}

	//getting env variables
	verifyToken, found := os.LookupEnv("verify_token")
	if !found {
		log.Fatal("environment variable verify_token not found")
	}
	appSecret, found := os.LookupEnv("app_secret")
	if !found {
		log.Fatal("envrionment variable app_secret not found")
	}

	//registering routes
	mux := http.NewServeMux()
	listener.Register(mux, verifyToken, appSecret)

	//starting server
	if err := Serve(mux); err != nil {
		log.Fatal(err)
	}
}
