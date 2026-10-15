package main

import (
	"encoding/json"
	"net/http"
	"net/url"

	. "github.com/ayushmantripathy/url-shortner/internal"
)

type createRedirectBody struct {
	Alias string `json:"alias"`
	RedirectsTo string `json:"redirectsTo"`
}

func createNewRedirect(w http.ResponseWriter, r *http.Request) {
	var body createRedirectBody
	json.NewDecoder(r.Body).Decode(&body)
	
	if len(body.Alias) < 3 {
		ReturnHTTPMessage(w, http.StatusBadRequest, "alias length cannot be less that 3")
		return
	}

	contents, err := url.Parse(body.RedirectsTo)
	if err != nil || contents.Scheme == "" || contents.Host == "" {
		ReturnHTTPMessage(w, http.StatusBadRequest, "redirectsTo should be a valid URL")
		return
	}

	_, err1 := Db.Exec(InsertRedirectQuery, body.Alias, body.RedirectsTo)
	if err1 != nil {
		ReturnHTTPMessage(w, http.StatusBadRequest, err1.Error())
		return
	}

	ReturnJSONResponse(w, http.StatusCreated).Encode(body)
}
