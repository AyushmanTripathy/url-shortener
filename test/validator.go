package test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

var master string = "http://localhost:8000"
var edge string = "http://localhost:8001"

type HttpTestcase struct {
	base string
	endpoint string
	method string
	reqBody string
	contentType string
	statusCode int
	response string
}

func Assert(t * testing.T, test HttpTestcase, a any, b any, expected string) {
	if a != b {
		t.Fatal(test.method, test.endpoint, expected, "expected", b, "received", a)
	}
}

func ValidateTestcase(t * testing.T, test HttpTestcase) {
	client := http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// This tells Go to stop following redirects and return the current response
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest(test.method, test.base + test.endpoint, bytes.NewReader([]byte(test.reqBody)))
	Assert(t, test, err, nil, "Request creation")

	resp, err := client.Do(req)
	Assert(t, test, err, nil, "Network request")

	Assert(t, test, resp.StatusCode, test.statusCode, "Status code")

	if len(test.response) == 0 {
		return
	}

	body, err := io.ReadAll(resp.Body)
	Assert(t, test, err, nil, "Response bytes")
	defer resp.Body.Close()

	Assert(t, test, strings.TrimSpace(string(body)), strings.TrimSpace(test.response), "Response body")
}

