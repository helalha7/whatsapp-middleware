package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/helalha7/whatsapp-middleware/internal/listener"
	"github.com/helalha7/whatsapp-middleware/internal/tenant"
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
	cfg := config{}

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
	value, found := os.LookupEnv("verify_token")
	if !found {
		log.Fatal("environment variable verify_token not found")
	}
	cfg.meta.verifyToken = value

	value, found = os.LookupEnv("app_secret")
	if !found {
		log.Fatal("envrionment variable app_secret not found")
	}
	cfg.meta.appSecret = value

	value, found = os.LookupEnv("db_dsn")
	if !found {
		log.Fatal("envrionment variable db dsn not found")
	}
	cfg.db.dsn = value

	return cfg
}

func main() {

	cfg := loadConfig()
	log.Println("loaded configurations")

	db, err := connectDB(cfg.db.dsn)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("connection to the database established")

	//registering routes
	mux := http.NewServeMux()
	listener.RegisterRoutes(
		mux,
		cfg.meta.verifyToken,
		cfg.meta.appSecret,
	)
	tenant.RegisterRoutes(
		mux,
		&tenant.Store{
			DB: db,
		},
	)

	//starting server
	if err := Serve(mux); err != nil {
		log.Fatal(err)
	}
}

func connectDB(
	dsn string,
) (*sql.DB, error) {
	db, err := sql.Open(
		"mysql",
		dsn,
	)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}
