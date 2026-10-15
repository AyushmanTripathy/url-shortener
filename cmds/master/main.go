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

	routes.HandleFunc("/", createNewRedirect).Methods("POST")

	http.ListenAndServe(":8080", routes)
}
