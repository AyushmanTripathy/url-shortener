package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
)

type createRedirectBody struct {
	Alias string `json:"alias"`
	RedirectsTo string `json:"redirectsTo"`
}

func Redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	log.Println("Redirect", vars["code"])

	rows, err := db.Query(getRedirectURLQuery, vars["code"])
	if err != nil {
		w.WriteHeader(500)
		log.Println("Redirect failed for code =", vars["code"])
		log.Println(err)
		return
	} 

	var redirectURL string
	if rows.Next() {
		rows.Scan(&redirectURL)
	}

	if len(redirectURL) < 2 {
		w.WriteHeader(404)
		fmt.Fprintln(w, "No redirect registered for code", vars["code"])
		return
	}

	http.Redirect(w, r, redirectURL, 302)
}

func CreateNewRedirect(w http.ResponseWriter, r *http.Request) {
	var body createRedirectBody
	json.NewDecoder(r.Body).Decode(&body)
	
	if len(body.Alias) < 3 {
		returnHTTPMessage(w, http.StatusBadRequest, "alias length cannot be less that 3")
		return
	}

	contents, err := url.Parse(body.RedirectsTo)
	if err != nil || contents.Scheme == "" || contents.Host == "" {
		returnHTTPMessage(w, http.StatusBadRequest, "redirectsTo should be a valid URL")
		return
	}

	_, err1 := db.Exec(insertRedirectQuery, body.Alias, body.RedirectsTo)
	if err1 != nil {
		returnHTTPMessage(w, http.StatusBadRequest, err1.Error())
		return
	}

	returnJSONResponse(w, http.StatusCreated).Encode(body)
}

