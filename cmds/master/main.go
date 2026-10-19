package main

import (
	"log"
	"net/http"
	"os"

	. "github.com/ayushmantripathy/url-shortner/internal"
	"github.com/gorilla/mux"
)

func main() {
	InitializeDBConnection()
	defer Db.Close()

	routes := mux.NewRouter()

	routes.HandleFunc("/", createNewRedirect).Methods("POST")

	port, found := os.LookupEnv("PORT")
	if !found {
		log.Fatal("PORT not found")
	}

	http.ListenAndServe(":" + port, routes)
}
