package main

import (
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type config struct {
	meta struct {
		appSecret   string
		verifyToken string
	}
	db struct {
		dsn string
	}
}

func loadConfig() config {
	environment := flag.String(
		"env",
		"development",
		"set the environment to production or development",
	)
	flag.Parse()

	if *environment == "development" {
		if err := godotenv.Load("./.env"); err != nil {
			log.Fatal(err)
		}
	}

	var cfg config
	cfg.meta.verifyToken = requiredEnvironmentVariable("verify_token")
	cfg.meta.appSecret = requiredEnvironmentVariable("app_secret")
	cfg.db.dsn = requiredEnvironmentVariable("db_dsn")

	return cfg
}

func requiredEnvironmentVariable(name string) string {
	value, found := os.LookupEnv(name)
	if !found {
		log.Fatalf("environment variable %s not found", name)
	}

	return value
}
