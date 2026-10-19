package main

import (
	"log"
	"net/http"
	"os"

	. "github.com/ayushmantripathy/url-shortner/internal"
	"github.com/gorilla/mux"
)

var redirectCache Cache

func main() {
	redirectCache.init(1000000)
	InitializeDBConnection()
	defer Db.Close()

	routes := mux.NewRouter()

	routes.HandleFunc("/{alias}", redirect).Methods("GET")

	port, found := os.LookupEnv("PORT")
	if !found {
		log.Fatal("PORT not found")
	}

	http.ListenAndServe(":" + port, routes)
}
