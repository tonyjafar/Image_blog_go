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
	list := []string{}
	if err == nil {
		c.MaxAge = cAge
		http.SetCookie(w, c)
	}
	if loggedIn(w, r) {
		test := &data
		test.loggedin = true
		rows, err := db.Query(
			`
			SELECT name FROM
    image_blog.images
ORDER BY created_at DESC
LIMIT 6
			`,
		)
		if err != nil {
			fmt.Println(err)
			return
		}
		var name string
		for rows.Next() {
			err := rows.Scan(&name)
			if err != nil {
				fmt.Println(err)
				return
			}
			list = append(list, name)
		}
	} else {
		test := &data
		test.loggedin = false
	}
	if len(list) == 0 {
		tpl.ExecuteTemplate(w, "index.gohtml", data.loggedin)
		return
	}
	tpl.ExecuteTemplate(w, "index.gohtml", list)
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
	// TODO : Add Pagination.
	c, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	list := []string{}
	c.MaxAge = cAge
	http.SetCookie(w, c)
	test := &data
	test.loggedin = true
	rows, err := db.Query(
		`
		SELECT name FROM
image_blog.images
ORDER BY created_at DESC
		`,
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	var name string
	for rows.Next() {
		err := rows.Scan(&name)
		if err != nil {
			fmt.Println(err)
			return
		}
		list = append(list, name)
	}
	if len(list) == 0 {
		tpl.ExecuteTemplate(w, "images.gohtml", data.loggedin)
		return
	}
	tpl.ExecuteTemplate(w, "images.gohtml", list)
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
		errors := make(map[string]bool)
		tn := time.Now()
		l := r.FormValue("location")
		d := r.FormValue("description")
		fhs := r.MultipartForm.File["nf"]
		for _, fhm := range fhs {
			mf, err := fhm.Open()
			if err != nil {
				errors["fileError"] = true
				tpl.ExecuteTemplate(w, "uplimage.gohtml", errors)
				return
			}
			defer mf.Close()
			s := fhm.Size
			ext := strings.Split(fhm.Filename, ".")[1]
			h := sha1.New()
			io.Copy(h, mf)
			n := fmt.Sprintf("%x", h.Sum(nil)) + "." + ext
			wd, err := os.Getwd()
			if err != nil {
				errors["fileError"] = true
				tpl.ExecuteTemplate(w, "uplimage.gohtml", errors)
				return
			}
			path := filepath.Join(wd, "data", n)
			nf, err := os.Create(path)
			if err != nil {
				errors["fileError"] = true
				tpl.ExecuteTemplate(w, "uplimage.gohtml", errors)
				return
			}
			defer nf.Close()
			mf.Seek(0, 0)
			io.Copy(nf, mf)
			image := &Image{n, l, s, tn, d}
			i := Save(image)
			if i != nil {
				te, err := os.Open(path)
				if err == nil {
					defer te.Close()
					os.Remove(path)
				}
				errors["fileError"] = true
				tpl.ExecuteTemplate(w, "uplimage.gohtml", errors)
				return
			}
		}
		errors["fileError"] = false
	}
	test := &data
	test.loggedin = true
	tpl.ExecuteTemplate(w, "uplimage.gohtml", data.loggedin)
}

func handleFileServer(dir, prefix string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(dir))
	realHandler := http.StripPrefix(prefix, fs).ServeHTTP
	return func(w http.ResponseWriter, r *http.Request) {
		if loggedIn(w, r) {
			realHandler(w, r)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

}

func search(w http.ResponseWriter, r *http.Request) {
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
	list := []string{}
	test := &data
	test.loggedin = true
	//SELECT * FROM items WHERE items.xml LIKE '%123456%'
	if r.Method == http.MethodPost {
		s := r.FormValue("search")
		if s == "" {
			tpl.ExecuteTemplate(w, "search.gohtml", data.loggedin)
			return
		}
		newQuery := "%" + s + "%"

		rows, err := db.Query(
			`
		SELECT name FROM
image_blog.images
WHERE description LIKE ?
		`, newQuery,
		)
		if err != nil {
			fmt.Println(err)
			return
		}
		var name string
		for rows.Next() {
			err := rows.Scan(&name)
			if err != nil {
				fmt.Println(err)
				return
			}
			list = append(list, name)
		}
	}
	if len(list) == 0 {
		tpl.ExecuteTemplate(w, "search.gohtml", data.loggedin)
		return
	}
	tpl.ExecuteTemplate(w, "search.gohtml", list)
}
