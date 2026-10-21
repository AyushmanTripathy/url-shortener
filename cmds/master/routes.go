package main

import (
	"encoding/json"
	"net/http"
	"net/url"

	. "github.com/ayushmantripathy/url-shortner/internal"
)

type redirectBody struct {
	Alias       string `json:"alias"`
	RedirectsTo string `json:"redirectsTo"`
}

func (body *redirectBody) validate() (string, bool) {
	if len(body.Alias) < 3 {
		return "alias length cannot be less that 3", true
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

	if _, err := Db.Exec(InsertRedirectQuery, body.Alias, body.RedirectsTo); err != nil {
		ReturnHTTPMessage(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	ReturnJSONResponse(w, http.StatusCreated).Encode(body)
}

func updateExistingRedirect(w http.ResponseWriter, r *http.Request) {
	var body redirectBody
	json.NewDecoder(r.Body).Decode(&body)

	if msg, invalid := body.validate(); invalid {
		ReturnHTTPMessage(w, http.StatusBadRequest, msg)
		return
	}

	if _, err := Db.Exec(UpdateRedirectURLQuery, body.Alias, body.RedirectsTo); err != nil {
		ReturnHTTPMessage(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// Return Accepted 202 if alias updation failed
	if natsConn.Emit("alias.updates", body.Alias) != nil {
		ReturnJSONResponse(w, http.StatusAccepted).Encode(body)
	} else {
		ReturnJSONResponse(w, http.StatusCreated).Encode(body)
	}
}
