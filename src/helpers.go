package main

import (
	"encoding/json"
	"net/http"
)

type httpError struct {
	Message string `json:"message"`
}

func returnHTTPMessage(w http.ResponseWriter, statuscode int, message string) {
	returnJSONResponse(w, statuscode).Encode(httpError{message})
}

func returnJSONResponse(w http.ResponseWriter, statuscode int) *json.Encoder {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statuscode)
	return json.NewEncoder(w)
}
