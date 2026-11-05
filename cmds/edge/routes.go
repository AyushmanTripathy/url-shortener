package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func fetchRedirect(alias string) (string, bool, error) {
	cached, found, valid := redirectCache.Get(alias)
	if found {
		return cached, valid, nil
	}

	resp, err := http.Get(masterUrl + alias)
	if err != nil {
		return "", false, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 503 {
		return "", false, errors.New("503 from master for " + alias)
	}

	if resp.StatusCode == 404 {
		redirectCache.Set(alias, "")
		return "", false, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false, err
	}

	redirectURL := string(body)
	redirectCache.Set(alias, redirectURL)
	return redirectURL, true, nil
}

func redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	defer redirectLogs.pushRedirectLog(r, vars["alias"])

	url, redirectPresent, err := fetchRedirect(vars["alias"])

	if err != nil {
		w.WriteHeader(503)
		log.Println("Fetching redirect failed", vars["alias"], err)
		fmt.Fprintln(w, "Unable to fetch alias", vars["alias"])
		return
	}

	if !redirectPresent {
		w.WriteHeader(404)
		fmt.Fprintln(w, "No redirect registered for alias", vars["alias"])
		return
	}

	http.Redirect(w, r, url, 302)
}
