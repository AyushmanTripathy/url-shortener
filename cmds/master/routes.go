package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"unicode"

	. "github.com/ayushmantripathy/url-shortner/internal"
	"github.com/gorilla/mux"
)

type redirectBody struct {
	Alias       string `json:"alias"`
	RedirectsTo string `json:"redirectsTo"`
}

func (body *redirectBody) validate() (string, bool) {
	if len(body.Alias) < 3 {
		return "alias length cannot be less that 3", true
	}

	for _, r := range body.Alias {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return "alias can only be unicode letters or digits", true
		}
	}

	contents, err := url.Parse(body.RedirectsTo)
	if err != nil || contents.Scheme == "" || contents.Host == "" {
		return "redirectsTo should be a valid URL", true
	}

	return "", false
}

func createNewRedirect(w http.ResponseWriter, r *http.Request) {
	var body redirectBody
	json.NewDecoder(r.Body).Decode(&body)

	if msg, invalid := body.validate(); invalid {
		ReturnHTTPMessage(w, http.StatusBadRequest, msg)
		return
	}

	if err := InsertRedirectEntry(body.Alias, body.RedirectsTo); err != nil {
		ReturnHTTPMessage(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	redirectCache.Set(body.Alias, body.RedirectsTo)
	ReturnJSONResponse(w, http.StatusCreated).Encode(body)
}

func updateExistingRedirect(w http.ResponseWriter, r *http.Request) {
	var body redirectBody
	json.NewDecoder(r.Body).Decode(&body)

	if msg, invalid := body.validate(); invalid {
		ReturnHTTPMessage(w, http.StatusBadRequest, msg)
		return
	}

	if err := UpdateRedirectEntry(body.Alias, body.RedirectsTo); err != nil {
		ReturnHTTPMessage(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	redirectCache.Set(body.Alias, body.RedirectsTo)
	// Return Accepted 202 if alias updation failed
	if natsConn.Emit("alias.updates", []byte(body.Alias)) != nil {
		ReturnJSONResponse(w, http.StatusAccepted).Encode(body)
	} else {
		ReturnJSONResponse(w, http.StatusOK).Encode(body)
	}
}

func fetchRedirect(alias string) (string, bool) {
	cached, found := redirectCache.Get(alias)
	if found {
		return cached, true
	}

	rows, err := Db.Query(GetRedirectURLQuery, alias)
	if err != nil {
		return "", false
	}

	var redirectURL string
	if rows.Next() {
		rows.Scan(&redirectURL)
	}

	redirectCache.Set(alias, redirectURL)
	if len(redirectURL) < 2 {
		return "", false
	}

	return redirectURL, true
}

func getRedirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	url, found := fetchRedirect(vars["alias"])
	if found {
		w.WriteHeader(200)
		fmt.Fprint(w, url)
	} else {
		w.WriteHeader(404)
	}
}
