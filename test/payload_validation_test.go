package test

import (
	"crypto/rand"
	"fmt"
	"testing"
)

func TestPayloadValidation(t * testing.T) {
	alias := rand.Text()[0:5]

	payloads := []string{
		`{}`,
		`{`,
		fmt.Sprintf(`{"alias": "%s-"}`, alias),
		fmt.Sprintf(`{"alias": "%s-", "redirectsTo": "http://%s.com"}`, alias, alias),
		fmt.Sprintf(`{"alias": "", "redirectsTo": "https://%s.com"}`, alias),
		fmt.Sprintf(`{"alias": "", "redirectsTo": "htps://%s.com"}`, alias),
		fmt.Sprintf(`{"alias": "%s", "redirectsTo": "%s"}`, alias, alias),
		fmt.Sprintf(`{"alias": "%s", "redirectsTo": "%s"}`, alias[0:2], alias),
	}

	for _, value := range payloads {
		ValidateTestcase(t, HttpTestcase{
			base: master,
			endpoint: "/",
			statusCode: 400,
			method: "PATCH",
			reqBody: value,
		})

		ValidateTestcase(t, HttpTestcase{
			base: master,
			endpoint: "/",
			statusCode: 400,
			method: "POST",
			reqBody: value,
		})
	}
}
