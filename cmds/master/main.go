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
	natsConn.EnsureStream("LOGS", []string{"logs.redirects"})
	natsConn.EnsureConsumer("master", "LOGS", "logs.redirects")
	natsConn.Receive("master", "LOGS", handleRedirectLogs)
	defer natsConn.CloseReceiver()

	routes := mux.NewRouter()

	routes.HandleFunc("/", createNewRedirect).Methods("POST")
	routes.HandleFunc("/", updateExistingRedirect).Methods("PATCH")

	port, found := os.LookupEnv("PORT")
	if !found {
		log.Fatal("PORT not found")
	}

	http.ListenAndServe(":"+port, routes)
}
