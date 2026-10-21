package main

import (
	"log"
	"net/http"
	"os"

	. "github.com/ayushmantripathy/url-shortner/internal"
	"github.com/gorilla/mux"
)

var natsConn NATSConnection

func main() {
	InitializeDBConnection()
	defer Db.Close()

	natsConn.Init()

	routes := mux.NewRouter()

	routes.HandleFunc("/", createNewRedirect).Methods("POST")
	routes.HandleFunc("/", updateExistingRedirect).Methods("PATCH")

	port, found := os.LookupEnv("PORT")
	if !found {
		log.Fatal("PORT not found")
	}

	http.ListenAndServe(":"+port, routes)
}
