package main

import (
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

type App struct {
	rdb          *redis.Client
	pr           *ProductRepository
	rateLimit    int64
	rateLimitKey string
}

func (a App) storeAdd(w http.ResponseWriter, r *http.Request) {
	p := new(Product)
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

	json.NewEncoder(w).Encode(res)
}

func (a App) storeOrder(w http.ResponseWriter, r *http.Request) {
	p := new(Product)
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

	json.NewEncoder(w).Encode(res)
}

func (a App) storeGetProduct(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	productId := params["product_id"]
	p, pgErr := a.pr.FindById(productId)

	if pgErr != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(p)
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
	})
}

func (a App) responseHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
