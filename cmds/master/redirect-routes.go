package main

import (
	"fmt"
	"log"
	"net/http"

	. "github.com/ayushmantripathy/url-shortner/internal"
	"github.com/gorilla/mux"
)

func fetchRedirect(alias string) (string, bool, error) {
	cached, found, valid := redirectCache.Get(alias)
	if found {
		return cached, valid, nil
	}

	rows, err := Db.Query(GetRedirectURLQuery, alias)
	if err != nil {
		return "", false, err
	}
	defer rows.Close()

	var redirectURL string
	if rows.Next() {
		rows.Scan(&redirectURL)
	}

	redirectCache.Set(alias, redirectURL)
	if len(redirectURL) < 2 {
		return "", false, nil
	}

	redirectCache.Set(alias, redirectURL)
	return redirectURL, true, nil
}

func getRedirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url, redirectPresent, err := fetchRedirect(vars["alias"])

	if err != nil {
		w.WriteHeader(503)
		log.Println("Fetching redirect failed", vars["alias"], err)
		return
	}

	if !redirectPresent {
		w.WriteHeader(404)
		return
	}

	w.WriteHeader(200)
	fmt.Fprint(w, url)
}
