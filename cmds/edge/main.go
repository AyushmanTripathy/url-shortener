package main

import (
	"log"
	"net/http"
	"os"

	. "github.com/ayushmantripathy/url-shortner/internal"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go/jetstream"
)

var redirectCache Cache
var natsConn NATSConnection
var instanceName string

func main() {
	if hostname, err := os.Hostname(); err != nil {
		log.Fatal("Failed to deduce instanceName")
	} else {
		instanceName = hostname
		log.Println("Name: ", instanceName)
	}

	redirectCache.init(1000000)
	InitializeDBConnection()
	defer Db.Close()

	natsConn.Init()
	natsConn.EnsureStream("ALIAS_UPDATIONS", []string{"alias.updates"})
	natsConn.EnsureConsumer(instanceName, "ALIAS_UPDATIONS", "alias.updates")
	natsConn.Receive(instanceName, "ALIAS_UPDATIONS", func(msg jetstream.Msg) {
		redirectCache.invalidate(string(msg.Data()))
		msg.Ack()
	})
	defer natsConn.CloseReceiver()

	routes := mux.NewRouter()

	routes.HandleFunc("/{alias}", redirect).Methods("GET")

	port, found := os.LookupEnv("PORT")
	if !found {
		log.Fatal("PORT not found")
	}

	http.ListenAndServe(":"+port, routes)
}
