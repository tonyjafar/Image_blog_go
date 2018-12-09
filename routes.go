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

	"github.com/disintegration/imaging"
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

func index(w http.ResponseWriter, r *http.Request) {
	SentData := &Data
	c, err := r.Cookie("session")
	List := []string{}
	if err == nil {
		c.MaxAge = cAge
		http.SetCookie(w, c)
	}
	if loggedIn(w, r) {
		SentData.Loggedin = true
		rows, err := db.Query(
			`
			SELECT name FROM
    image_blog.images
ORDER BY created_at DESC
LIMIT 6
			`,
		)
		if err != nil {
			log.Error(err.Error())
			return
		}
		var name string
		for rows.Next() {
			err := rows.Scan(&name)
			if err != nil {
				log.Error(err.Error())
				return
			}
			List = append(List, name)
		}
		SentData.List = List
	} else {
		SentData.Loggedin = false
	}
	tpl.ExecuteTemplate(w, "index.gohtml", SentData)
}

func login(w http.ResponseWriter, r *http.Request) {
	SentData := &Data
	SentData.UserError = false
	if loggedIn(w, r) {
		http.Redirect(w, r, "/images", http.StatusSeeOther)
		return
	}
	if strings.HasSuffix(r.RequestURI, ".css") {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	}
	if r.Method == http.MethodPost {
		un := r.FormValue("username")
		p := r.FormValue("password")
		var name string
		var pass string
		row1 := db.QueryRow("select username from image_blog.Users where username=?", un).Scan(&name)
		if row1 != nil {
			SentData.UserError = true
		}
		row2 := db.QueryRow("select password from image_blog.Users where username=?", un).Scan(&pass)
		if row2 != nil {
			SentData.UserError = true
		}
		bp := []byte(p)
		st2 := []byte(pass)
		pe := bcrypt.CompareHashAndPassword(st2, bp)
		blocked, err := getAndUpdateRetry(un)
		if err != nil {
			SentData.UserError = true
			log.Fatalf("Unable to connect to databese to determine the status of the User - %s", err.Error())
			tpl.ExecuteTemplate(w, "signin.gohtml", SentData)
			return
		}
		if un == name && pe == nil && !blocked {
			db.Exec(
				`
				update image_blog.Users SET retry = 0 WHERE username = ?
				`,
				un,
			)
			s, _ := uuid.NewV4()
			cookieValue := s.String() + "," + un
			c := &http.Cookie{
				Name:   "session",
				Value:  cookieValue,
				MaxAge: cAge,
			}
			http.SetCookie(w, c)
			err := updateUserSession(s.String(), un)
			if err != nil {
				SentData.UserError = true
				log.Fatalf("Can not update the user session after login - %s", err.Error())
				tpl.ExecuteTemplate(w, "signin.gohtml", SentData)
				return
			}
			log.Infof("User %s logged in", un)
			http.Redirect(w, r, "/images", http.StatusSeeOther)
			return
		} else {
			log.Error("Authentication Failed!!")
			SentData.UserError = true
			tpl.ExecuteTemplate(w, "signin.gohtml", SentData)
			return
		}
	}
	tpl.ExecuteTemplate(w, "signin.gohtml", SentData)
}

func images(w http.ResponseWriter, r *http.Request) {
	SentData := &Data
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	if strings.HasSuffix(r.RequestURI, ".css") {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	}
	List := []string{}
	c.MaxAge = cAge
	http.SetCookie(w, c)
	SentData.Loggedin = true
	rows, err := db.Query(
		`
		SELECT name FROM
image_blog.images
ORDER BY created_at DESC
		`,
	)
	if err != nil {
		log.Error(err.Error())
		return
	}
	var name string
	for rows.Next() {
		err := rows.Scan(&name)
		if err != nil {
			log.Error(err.Error())
			return
		}
		List = append(List, name)
	}
	SentData.List = List
	pageIt(w, &SentData.MyVar, r, List, false)
	tpl.ExecuteTemplate(w, "images.gohtml", SentData)
}

func signout(w http.ResponseWriter, r *http.Request) {
	SentData := &Data
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
	username := strings.Split(c.Value, ",")[1]
	SentData.Loggedin = false
	db.Exec(
		`
		update image_blog.Users SET session = null WHERE username = ?
		`,
		username,
	)
	log.Infof("User %s logged out", username)
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return
}

func addImage(w http.ResponseWriter, r *http.Request) {
	SentData := &Data
	SentData.ErrorFile = FileError{}
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
		tn := time.Now()
		l := r.FormValue("location")
		d := r.FormValue("description")
		fhs := r.MultipartForm.File["nf"]
		for _, fhm := range fhs {
			mf, err := fhm.Open()
			if err != nil {
				SentData.ErrorFile.IsError = true
				SentData.ErrorFile.ErrorType = err.Error()
				tpl.ExecuteTemplate(w, "uplimage.gohtml", SentData)
				return
			}
			defer mf.Close()
			s := fhm.Size
			if !checkFileName(fhm.Filename) {
				SentData.ErrorFile.IsError = true
				SentData.ErrorFile.ErrorType = "File name does not contian extension"
				log.Errorf("File %s name error", fhm.Filename)
				tpl.ExecuteTemplate(w, "uplimage.gohtml", SentData)
				return
			}
			ext := strings.Split(fhm.Filename, ".")[1]
			h := sha1.New()
			io.Copy(h, mf)
			n := fmt.Sprintf("%x", h.Sum(nil)) + "." + ext
			wd, err := os.Getwd()
			if err != nil {
				log.Error(err.Error())
				SentData.ErrorFile.IsError = true
				SentData.ErrorFile.ErrorType = err.Error()
				tpl.ExecuteTemplate(w, "uplimage.gohtml", SentData)
				return
			}
			path := filepath.Join(wd, "data", n)
			nf, err := os.Create(path)
			defer nf.Close()
			if err != nil {
				log.Error(err.Error())
				SentData.ErrorFile.IsError = true
				SentData.ErrorFile.ErrorType = "Create Path"
				tpl.ExecuteTemplate(w, "uplimage.gohtml", SentData)
				return
			}
			mf.Seek(0, 0)
			io.Copy(nf, mf)
			image := &Image{n, l, s, tn, d}
			scrImage, err := imaging.Open("./data/" + image.Name)
			if err != nil {
				log.Error(err.Error())
				mf.Close()
				nf.Close()
				os.Remove(path)
				SentData.ErrorFile.IsError = true
				SentData.ErrorFile.ErrorType = err.Error()
				tpl.ExecuteTemplate(w, "uplimage.gohtml", SentData)
				return
			}
			i := Save(image)
			dstImage := imaging.Thumbnail(scrImage, widthThumbnail, widthThumbnail, imaging.Lanczos)
			destination := "./data/thumb/" + image.Name
			it := imaging.Save(dstImage, destination)
			if i != nil && it != nil {
				te, err := os.Open(path)
				if err == nil {
					defer te.Close()
					os.Remove(path)
				}
				SentData.ErrorFile.IsError = true
				SentData.ErrorFile.ErrorType = err.Error()
				tpl.ExecuteTemplate(w, "uplimage.gohtml", SentData)
				return
			}
		}
		SentData.ErrorFile.IsError = false
		SentData.ErrorFile.IsSucc = true
	}
	SentData.Loggedin = true
	tpl.ExecuteTemplate(w, "uplimage.gohtml", SentData)
}

func search(w http.ResponseWriter, r *http.Request) {
	SentData := &Data
	SentData.List = []string{}
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
	List := []string{}
	SentData.Loggedin = true
	var v bool
	if r.Method == http.MethodPost || strings.Contains(r.RequestURI, "page") || strings.Contains(r.RequestURI, "all") {
		video := &v
		r.ParseForm()
		s := r.FormValue("search")
		rad := r.Form["optradio"]
		if s == "" {
			tpl.ExecuteTemplate(w, "search.gohtml", SentData)
			return
		}
		newQuery := "%" + s + "%"
		var query string
		if len(rad) > 0 {
			if rad[0] == "video" {
				query = `SELECT name FROM
				image_blog.videos
				WHERE description LIKE ?
				ORDER BY created_at DESC`
				*video = true
			} else {
				query = `SELECT name FROM
				image_blog.images
				WHERE description LIKE ?
				ORDER BY created_at DESC`
				*video = false
			}
		} else {
			query = `SELECT name FROM
				image_blog.images
				WHERE description LIKE ?
				ORDER BY created_at DESC`
			*video = false
		}

		rows, err := db.Query(query, newQuery)
		if err != nil {
			log.Error(err.Error())
			return
		}
		var name string
		for rows.Next() {
			err := rows.Scan(&name)
			if err != nil {
				log.Error(err.Error())
				return
			}
			List = append(List, name)
		}
		SentData.List = List
	}
	pageIt(w, &SentData.MyVar, r, SentData.List, v)
	tpl.ExecuteTemplate(w, "search.gohtml", &SentData)
	return

}

func addVideo(w http.ResponseWriter, r *http.Request) {
	SentData := &Data
	SentData.ErrorFile = FileError{}
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
		tn := time.Now()
		l := r.FormValue("location")
		d := r.FormValue("description")
		fhs := r.MultipartForm.File["nf"]
		for _, fhm := range fhs {
			mf, err := fhm.Open()
			if err != nil {
				log.Error(err.Error())
				SentData.ErrorFile.IsError = true
				SentData.ErrorFile.ErrorType = err.Error()
				tpl.ExecuteTemplate(w, "uploadvideo.gohtml", SentData)
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
				log.Error(err.Error())
				SentData.ErrorFile.IsError = true
				SentData.ErrorFile.ErrorType = err.Error()
				tpl.ExecuteTemplate(w, "uploadvideo.gohtml", SentData)
				return
			}
			path := filepath.Join(wd, "data/videos", n)
			nf, err := os.Create(path)
			if err != nil {
				log.Error(err.Error())
				SentData.ErrorFile.IsError = true
				SentData.ErrorFile.ErrorType = err.Error()
				tpl.ExecuteTemplate(w, "uploadvideo.gohtml", SentData)
				return
			}
			defer nf.Close()
			mf.Seek(0, 0)
			io.Copy(nf, mf)
			video := &Video{n, l, s, tn, d}
			i := SaveV(video)
			if i != nil {
				te, err := os.Open(path)
				if err == nil {
					defer te.Close()
					os.Remove(path)
				}
				SentData.ErrorFile.IsError = true
				SentData.ErrorFile.ErrorType = err.Error()
				tpl.ExecuteTemplate(w, "uploadvideo.gohtml", SentData)
				return
			}
		}
		SentData.ErrorFile.IsError = false
		SentData.ErrorFile.IsSucc = true
	}
	SentData.Loggedin = true
	tpl.ExecuteTemplate(w, "uploadvideo.gohtml", SentData)
}

func videos(w http.ResponseWriter, r *http.Request) {
	SentData := &Data
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	List := []string{}
	c.MaxAge = cAge
	http.SetCookie(w, c)
	SentData.Loggedin = true
	rows, err := db.Query(
		`
		SELECT name FROM
image_blog.videos
ORDER BY created_at DESC
		`,
	)
	if err != nil {
		log.Error(err.Error())
		return
	}
	var name string
	for rows.Next() {
		err := rows.Scan(&name)
		if err != nil {
			log.Error(err.Error())
			return
		}
		List = append(List, name)
	}
	SentData.List = List
	pageIt(w, &SentData.MyVar, r, List, true)
	tpl.ExecuteTemplate(w, "videos.gohtml", SentData)

}
