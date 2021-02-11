package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-pg/pg/v10"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/qctest/repo/model"
	"github.com/qctest/repo/repository"
	"log"
	"net/http"
	"os"
)

type App struct {
	rdb          *redis.Client
	pr           *repository.ProductRepository
	rateLimit    int64
	rateLimitKey string
}

func (a App) storeAdd(w http.ResponseWriter, r *http.Request) {
	p := new(model.Product)
	err := json.NewDecoder(r.Body).Decode(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, pgErr := a.pr.Store(p)

	if pgErr != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%+v", res)
}

func (a App) storeOrder(w http.ResponseWriter, r *http.Request) {
	p := new(model.Product)
	err := json.NewDecoder(r.Body).Decode(p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, pgErr := a.pr.Order(p)

	if pgErr != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if res == 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "%+v", res)
}

func (a App) storeGetProduct(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productId := params["product_id"]
	p, err := a.pr.FindById(productId)

	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "%+v", p)
}

func (a App) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		currRate, rdbErr := a.rdb.Incr(ctx, a.rateLimitKey).Result()

		if rdbErr != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			log.Printf("%+v", rdbErr.Error())
			return
		}

		if currRate > a.rateLimit {
			a.rdb.Decr(ctx, a.rateLimitKey)
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		log.Printf("Rate hit: %+v", currRate)

		next.ServeHTTP(w, r)

		a.rdb.Decr(r.Context(), a.rateLimitKey)
		w.Header().Set("Content-Type", "application/json")
	})
}

func (a App) responseHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		w.Header().Set("Content-Type", "application/json")
	})
}

func main() {
	r := mux.NewRouter()

	pgdb := ProvidePostgresConnect()
	defer pgdb.Close()

	rdb := ProvideRedisClient()
	defer rdb.Close()

	app := new(App)
	app.pr = repository.NewProductRepository(pgdb)
	app.rdb = rdb
	app.rateLimit = 3
	app.rateLimitKey = "rlk"

	rdb.Set(context.Background(), app.rateLimitKey, 0, 0)

	so := r.PathPrefix("/store/order").Methods("POST").Subrouter()
	so.HandleFunc("", app.storeOrder)
	so.Use(app.rateLimitMiddleware)

	r.HandleFunc("/store/add", app.storeAdd).Methods("POST")
	r.HandleFunc("/store/{product_id:[0-9]+}", app.storeGetProduct).Methods("GET")

	r.Use(app.responseHeadersMiddleware)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
		log.Printf("Defaulting to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	log.Printf("Open http://localhost see your actual port in a docker-compose file")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), r))
}

func ProvidePostgresConnect() *pg.DB {
	pgdb := pg.Connect(&pg.Options{
		Addr:     "postgres:5432",
		User:     "postgres",
		Password: "pass",
		Database: "qctest",
	})
	return pgdb
}

func ProvideRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "redis-server:6379",
		Password: "",
		DB:       0,
	})
}
