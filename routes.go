package main

import (
	"net/http"

	"github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

func index(w http.ResponseWriter, r *http.Request) {
	x := loggedIn(w, r)
	tpl.ExecuteTemplate(w, "index.gohtml", x)
}

func login(w http.ResponseWriter, r *http.Request) {

	if loggedIn(w, r) {
		http.Redirect(w, r, "/images", http.StatusSeeOther)
		return
	}

	var userData struct {
		userPassErr bool
	}
	if r.Method == http.MethodPost {
		un := r.FormValue("username")
		p := r.FormValue("password")
		var name string
		var pass string
		row1 := db.QueryRow("select username from image_blog.users where username=?", un).Scan(&name)
		if row1 != nil {
			userData.userPassErr = true
		}
		hp, _ := bcrypt.GenerateFromPassword([]byte(p), bcrypt.MinCost)
		st := string(hp)
		row2 := db.QueryRow("select password from image_blog.users where password=?", st).Scan(&pass)
		if row2 != nil {
			userData.userPassErr = true
		}
		bp := []byte(p)
		pe := bcrypt.CompareHashAndPassword(hp, bp)
		if un == name && pe == nil {
			s, _ := uuid.NewV4()
			c := &http.Cookie{
				Name:   "session",
				Value:  s.String(),
				MaxAge: 30,
			}
			http.SetCookie(w, c)
		}
	}
	tpl.ExecuteTemplate(w, "signin.gohtml", userData)
}
