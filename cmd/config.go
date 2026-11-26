package main

import (
	"flag"
	"log"
	"os"
	"time"

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
	ai struct {
		sessionIdleTimeout time.Duration
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
	cfg.ai.sessionIdleTimeout = durationEnvironmentVariable(
		"ai_session_idle_timeout",
		10*time.Minute,
	)

	return cfg
}

func durationEnvironmentVariable(name string, defaultValue time.Duration) time.Duration {
	value, found := os.LookupEnv(name)
	if !found {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		log.Fatalf("environment variable %s must be a positive duration", name)
	}
	return duration
}

func requiredEnvironmentVariable(name string) string {
	value, found := os.LookupEnv(name)
	if !found {
		log.Fatalf("environment variable %s not found", name)
	}

	return value
}
