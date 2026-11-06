package test

import (
	"crypto/rand"
	"fmt"
	"testing"
)

func TestNegativeCaching(t * testing.T) {
	alias := rand.Text()[0:5]

	ValidateTestcase(t, HttpTestcase{
		base: edge,
		method: "GET",
		endpoint: fmt.Sprintf(`/%s`, alias),
		statusCode: 404,
	})

	ValidateTestcase(t, HttpTestcase{
		base: master,
		method: "POST",
		endpoint: "/",
		reqBody: fmt.Sprintf(`{"alias": "%s", "redirectsTo": "https://%s.com"}`, alias, alias),
		statusCode: 201,
	})

	ValidateTestcase(t, HttpTestcase{
		base: edge,
		method: "GET",
		endpoint: fmt.Sprintf(`/%s`, alias),
		statusCode: 302,
		response: fmt.Sprintf("<a href=\"https://%s.com\">Found</a>.\n", alias),
	})
}

func TestRedirectionFlow(t * testing.T) {
	alias := rand.Text()[0:5]
	ValidateTestcase(t, HttpTestcase{
		base: master,
		method: "POST",
		endpoint: "/",
		reqBody: fmt.Sprintf(`{"alias": "%s", "redirectsTo": "https://%s.com"}`, alias, alias),
		statusCode: 201,
	})

	ValidateTestcase(t, HttpTestcase{
		base: edge,
		method: "GET",
		endpoint: fmt.Sprintf(`/%s`, alias),
		statusCode: 302,
		response: fmt.Sprintf("<a href=\"https://%s.com\">Found</a>.\n", alias),
	})

	ValidateTestcase(t, HttpTestcase{
		base: master,
		method: "PATCH",
		endpoint: "/",
		reqBody: fmt.Sprintf(`{"alias": "%s", "redirectsTo": "https://%s1.com"}`, alias, alias),
		statusCode: 200,
	})

	ValidateTestcase(t, HttpTestcase{
		base: edge,
		method: "GET",
		endpoint: fmt.Sprintf(`/%s`, alias),
		statusCode: 302,
		response: fmt.Sprintf("<a href=\"https://%s1.com\">Found</a>.\n", alias),
	})
}
