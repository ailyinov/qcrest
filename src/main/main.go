package main

import (
	"context"
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"strconv"
)

const (
	RateKey   = "rlk"
	RateLimit = 3
)

func main() {
	r := mux.NewRouter()

	pgdb := ProvidePostgresConnect()
	defer pgdb.Close()

	rdb := ProvideRedisClient()
	defer rdb.Close()

	app := new(App)
	app.pr = NewProductRepository(pgdb)
	app.rdb = rdb
	app.rateLimit = RateLimit
	app.rateLimitKey = RateKey

	rdb.Set(context.Background(), app.rateLimitKey, 0, 0)

	r.Use(app.responseHeadersMiddleware)

	so := r.PathPrefix("/store/order").Methods("POST").Subrouter()
	so.HandleFunc("", app.storeOrder)
	so.Use(app.rateLimitMiddleware)

	r.HandleFunc("/store/add", app.storeAdd).Methods("POST")
	r.HandleFunc("/store/{product_id:[0-9]+}", app.storeGetProduct).Methods("GET")

	port := os.Getenv("CONTAINER_PORT")
	log.Printf("Defaulting to port %s", port)

	log.Printf("Listening on port %s", port)
	log.Printf("Open http://localhost see your actual port in a docker-compose file")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), r))
}

func ProvidePostgresConnect() *pg.DB {
	pgdb := pg.Connect(&pg.Options{
		Addr:     "postgres:" + os.Getenv("PG_PORT"),
		User:     os.Getenv("PG_USER"),
		Password: os.Getenv("PG_PASSWORD"),
		Database: os.Getenv("PG_DB"),
	})
	return pgdb
}

func ProvideRedisClient() *redis.Client {
	db, _ := strconv.Atoi(os.Getenv("REDIS_DB"))

	return redis.NewClient(&redis.Options{
		Addr: "redis-server:" + os.Getenv("REDIS_PORT"),
		DB:   db,
	})
}
