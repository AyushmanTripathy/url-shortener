package internal

import (
	"encoding/json"
	"net/http"
)

type RedirectLogHeader struct {
	Count int32  `json:"count"`
	Edge  string `json:"edge"`
}

type RedirectLog struct {
	Ip        string `json:"ip"`
	UserAgent string `json:"userAgent"`
	Code      string `json:"code"`
}

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
