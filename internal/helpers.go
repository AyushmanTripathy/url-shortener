package internal

import (
	"encoding/json"
	"net/http"
)

type httpError struct {
	Message string `json:"message"`
}

func ReturnHTTPMessage(w http.ResponseWriter, statuscode int, message string) {
	ReturnJSONResponse(w, statuscode).Encode(httpError{message})
}

func ReturnJSONResponse(w http.ResponseWriter, statuscode int) *json.Encoder {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statuscode)
	return json.NewEncoder(w)
}
