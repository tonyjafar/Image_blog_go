package main

import (
	"fmt"
	"net/http"

	"github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

func index(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session")
	if err == nil {
		c.MaxAge = cAge
		http.SetCookie(w, c)
	}
	if loggedIn(w, r) {
		test := &data
		test.loggedin = true
	} else {
		test := &data
		test.loggedin = false
	}
	tpl.ExecuteTemplate(w, "index.gohtml", data.loggedin)
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
		row1 := db.QueryRow("select username from image_blog.Users where username=?", un).Scan(&name)
		if row1 != nil {
			userData.userPassErr = true
			fmt.Println(row1)
		}
		row2 := db.QueryRow("select password from image_blog.Users where username=?", un).Scan(&pass)
		if row2 != nil {
			userData.userPassErr = true
			fmt.Println(row2)
		}
		bp := []byte(p)
		st2 := []byte(pass)
		pe := bcrypt.CompareHashAndPassword(st2, bp)
		if un == name && pe == nil {
			s, _ := uuid.NewV4()
			c := &http.Cookie{
				Name:   "session",
				Value:  s.String(),
				MaxAge: cAge,
			}
			http.SetCookie(w, c)
			http.Redirect(w, r, "/images", http.StatusSeeOther)
			return
		} else {
			userData.userPassErr = true
			tpl.ExecuteTemplate(w, "signin.gohtml", userData)
			return
		}
	}
	tpl.ExecuteTemplate(w, "signin.gohtml", nil)
}

func images(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	// TODO : implement upload/view
	c, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c.MaxAge = cAge
	http.SetCookie(w, c)
	test := &data
	test.loggedin = true
	tpl.ExecuteTemplate(w, "images.gohtml", data.loggedin)
}

func signout(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	c, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	c.MaxAge = -1
	http.SetCookie(w, c)
	test := &data
	test.loggedin = false
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return
}
