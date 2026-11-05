package main

import (
	"fmt"
	"io"
	"net/http"
	"github.com/gorilla/mux"
)

func redirect(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	defer redirectLogs.pushRedirectLog(r, vars["alias"])

	cached, found := redirectCache.Get(vars["alias"])
	if found {
		http.Redirect(w, r, cached, 302)
		return
	}

	resp, err := http.Get(masterUrl + vars["alias"])
	if err != nil {
		w.WriteHeader(503)
		fmt.Fprintln(w, "Unable to fetch alias", vars["alias"])
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		w.WriteHeader(404)
		fmt.Fprintln(w, "No redirect registered for alias", vars["alias"])
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintln(w, "Failed to parse entry for alias", vars["alias"])
		return
	}

	redirectURL := string(body)
	redirectCache.Set(vars["alias"], redirectURL)
	http.Redirect(w, r, redirectURL, 302)
}
