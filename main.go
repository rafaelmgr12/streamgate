package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func main() {

	r := chi.NewRouter()

	// Status endpoint
	r.Get("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("API Gateway is up and running"))
	})

	// Inicia o servidor HTTP
	log.Println("API Gateway running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))

}
