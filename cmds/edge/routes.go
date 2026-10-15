package main

import (
	"fmt"
	"log"
	"net/http"

	. "github.com/ayushmantripathy/url-shortner/internal"
	"github.com/gorilla/mux"
)

func redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Println("Redirect", vars["alias"])

	rows, err := Db.Query(GetRedirectURLQuery, vars["alias"])
	if err != nil {
		w.WriteHeader(500)
		log.Println("Redirect failed for alias =", vars["alias"])
		log.Println(err)
		return
	} 

	var redirectURL string
	if rows.Next() {
		rows.Scan(&redirectURL)
	}

	if len(redirectURL) < 2 {
		w.WriteHeader(404)
		fmt.Fprintln(w, "No redirect registered for alias", vars["alias"])
		return
	}

	http.Redirect(w, r, redirectURL, 302)
}
