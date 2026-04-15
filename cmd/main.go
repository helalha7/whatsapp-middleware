package main

import (
	"log"
	"whatsconnect/internal/infrastructure/webhook"

	"github.com/gin-gonic/gin"
)

func main() {

	router := gin.Default()

	webhook.RegisterRoutes(router, &webhook.Handler{})

	err := router.Run(":8080")
	if err != nil {
		log.Fatal(err)
	}
}
