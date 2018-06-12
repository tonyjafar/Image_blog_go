package main

import (
	"net/http"
)

func loggedIn(w http.ResponseWriter, r *http.Request) bool {
	c, err := r.Cookie("session")
	if err != nil {
		return false
	}
	var name string
	un := db.QueryRow("select username from image_blog.users where username=?", c.Value).Scan(&name)
	if un != nil {
		return false
	}
	if name == c.Value {
		c.MaxAge = cAge
		http.SetCookie(w, c)
		return true
	}
	return false
}
