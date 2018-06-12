package main

import (
	"net/http"
)

func loggedIn(r *http.Request) bool {
	_, err := r.Cookie("session")
	if err != nil {
		return false
	}
	return true
}

func getUser(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/sign_in", http.StatusSeeOther)
		return ""
	}
	return c.Value
}
