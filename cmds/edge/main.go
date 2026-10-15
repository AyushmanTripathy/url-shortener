package main

import (
	"net/http"

	"github.com/gorilla/mux"
	. "github.com/ayushmantripathy/url-shortner/internal"
)

func main() {
	InitializeDBConnection()
	defer Db.Close()

	routes := mux.NewRouter()

	routes.HandleFunc("/{alias}", redirect).Methods("GET")

	http.ListenAndServe(":8080", routes)
}
