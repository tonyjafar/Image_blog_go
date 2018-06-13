package main

import (
	"net/http"
)

func loggedIn(w http.ResponseWriter, r *http.Request) bool {
	_, err := r.Cookie("session")
	if err != nil {
		return false
	}
	return true
}
