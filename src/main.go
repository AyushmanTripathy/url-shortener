package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	InitializeDBConnection()
	defer db.Close()

	routes := mux.NewRouter()

	routes.HandleFunc("/{alias}", Redirect).Methods("GET")
	routes.HandleFunc("/", CreateNewRedirect).Methods("POST")

	http.ListenAndServe(":8080", routes)
}
