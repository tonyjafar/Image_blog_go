package main

import (
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

func addImage(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	c.MaxAge = cAge
	http.SetCookie(w, c)
	if r.Method == http.MethodPost {
		var errors struct {
			fileError bool
			dbError   bool
		}
		tn := time.Now()
		l := r.FormValue("location")
		d := r.FormValue("description")
		mf, fh, oerr := r.FormFile("nf")
		if oerr != nil {
			errors.fileError = true
			tpl.ExecuteTemplate(w, "uplimage", errors)
		}
		defer mf.Close()
		s := fh.Size
		ext := strings.Split(fh.Filename, ".")[1]
		h := sha1.New()
		io.Copy(h, mf)
		n := fmt.Sprintf("%x", h.Sum(nil)) + "." + ext
		wd, werr := os.Getwd()
		if werr != nil {
			errors.fileError = true
			tpl.ExecuteTemplate(w, "uplimage", errors)
		}
		path := filepath.Join(wd, "data", n)
		nf, herr := os.Create(path)
		if herr != nil {
			errors.fileError = true
			tpl.ExecuteTemplate(w, "uplimage", errors)
		}
		defer nf.Close()
		mf.Seek(0, 0)
		io.Copy(nf, mf)
		image := &Image{n, l, s, tn, d}
		i := Save(image)
		if i != nil {
			_, fr := os.Open(path)
			if fr != nil {
				os.Remove(path)
			}
			errors.dbError = true
			tpl.ExecuteTemplate(w, "uplimage", errors)
		}
		errors.dbError = false
		errors.fileError = false
	}
	tpl.ExecuteTemplate(w, "uplimage.gohtml", nil)
}
