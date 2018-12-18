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
			log.Errorf("Unable to connect to databese to determine the status of the User - %s", err.Error())
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
				log.Errorf("Can not update the user session after login - %s", err.Error())
				tpl.ExecuteTemplate(w, "signin.gohtml", SentData)
				return
			}
			log.Infof("User %s logged in", un)
			SentData.Username = un
			if isAdmin(un) {
				SentData.Admin = true
			}
			http.Redirect(w, r, "/images", http.StatusSeeOther)
			return
		} else {
			log.Errorf("Authentication Failed!! - using username %s", un)
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
	SentData.Admin = false
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
	username := strings.Split(c.Value, ",")[1]
	if !isAdmin(username) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
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
	username := strings.Split(c.Value, ",")[1]
	if !isAdmin(username) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
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

func admin(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, _ := r.Cookie("session")
	username := strings.Split(c.Value, ",")[1]
	if !isAdmin(username) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	SentData := &Data
	SentData.Admin = true
	var imageCount, videoCount, userCount, blockedUser string
	db.QueryRow("select count(*) from image_blog.images").Scan(&imageCount)
	db.QueryRow("select count(*) from image_blog.videos").Scan(&videoCount)
	db.QueryRow("select count(*) from image_blog.Users").Scan(&userCount)
	db.QueryRow("select count(*) from image_blog.Users where retry >= 5").Scan(&blockedUser)
	SentData.Statics.ImageCount = imageCount
	SentData.Statics.VideoCount = videoCount
	SentData.Statics.UserCount = userCount
	SentData.Statics.BlockedUser = blockedUser
	tpl.ExecuteTemplate(w, "index-admin.gohtml", SentData)
}

func imagesAdmin(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, _ := r.Cookie("session")
	username := strings.Split(c.Value, ",")[1]
	if !isAdmin(username) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	SentData := &Data
	SentData.Admin = true
	if r.Method == http.MethodPost {
		List := []Images{}
		r.ParseForm()
		s := r.FormValue("search-admin")
		newQuery := "%" + s + "%"
		query := `SELECT name,location,description, created_at FROM
				image_blog.images
				WHERE name LIKE ?
				`
		rows, err := db.Query(query, newQuery)
		if err != nil {
			log.Error(err.Error())
			return
		}
		var name, location, description, createdAt string
		for rows.Next() {
			err := rows.Scan(&name, &location, &description, &createdAt)
			if err != nil {
				log.Error(err.Error())
				return
			}
			image := Images{name, location, description, createdAt}
			List = append(List, image)
		}
		SentData.Statics.AdminSearch = true
		SentData.ImagesInfo = List
		tpl.ExecuteTemplate(w, "images-admin.gohtml", &SentData)
		return
	}
	query := `SELECT name,location,description, created_at FROM
				image_blog.images
				`
	rows, err := db.Query(query)
	if err != nil {
		log.Error(err.Error())
		return
	}
	List := []Images{}
	var name, location, description, createdAt string
	for rows.Next() {
		err := rows.Scan(&name, &location, &description, &createdAt)
		if err != nil {
			log.Error(err.Error())
			return
		}
		image := Images{name, location, description, createdAt}
		List = append(List, image)
	}
	SentData.ImagesInfo = List
	tpl.ExecuteTemplate(w, "images-admin.gohtml", &SentData)
	return

}

func imagesAdminDelete(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, _ := r.Cookie("session")
	username := strings.Split(c.Value, ",")[1]
	if !isAdmin(username) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodGet {
		r.ParseForm()
		name := r.FormValue("delete")
		if name != "" {
			db.Exec("delete from image_blog.images where name = ?", name)
		}
		http.Redirect(w, r, "/images-admin", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/images-admin", http.StatusSeeOther)
	return
}

func imagesAdminEdit(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, _ := r.Cookie("session")
	username := strings.Split(c.Value, ",")[1]
	if !isAdmin(username) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	SentData := &Data
	if r.Method == http.MethodGet {

		r.ParseForm()
		name := r.FormValue("name")
		if name != "" {
			query := `SELECT name,location,description, created_at FROM
				image_blog.images where name = ?
				`
			rows, err := db.Query(query, name)
			if err != nil {
				log.Error(err.Error())
				return
			}
			List := []Images{}
			var name, location, description, createdAt string
			for rows.Next() {
				err := rows.Scan(&name, &location, &description, &createdAt)
				if err != nil {
					log.Error(err.Error())
					return
				}
				image := Images{name, location, description, createdAt}
				List = append(List, image)
			}
			SentData.ImagesInfo = List
			tpl.ExecuteTemplate(w, "edit_image_admin.gohtml", &SentData)
			return

		}
	}
	if r.Method == http.MethodPost {
		r.ParseForm()
		name := r.FormValue("name")
		location := r.FormValue("location")
		description := r.FormValue("description")
		createdAt := r.FormValue("createdAt")
		db.Exec("update image_blog.images set location = ?, description = ?, created_at = ? where name = ?",
			location, description, createdAt, name)
		List := []Images{}
		image := Images{name, location, description, createdAt}
		List = append(List, image)
		SentData.ImagesInfo = List
		tpl.ExecuteTemplate(w, "edit_image_admin.gohtml", &SentData)
		return
	}

}

func videosAdmin(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, _ := r.Cookie("session")
	username := strings.Split(c.Value, ",")[1]
	if !isAdmin(username) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	SentData := &Data
	SentData.Admin = true
	if r.Method == http.MethodPost {
		List := []Images{}
		r.ParseForm()
		s := r.FormValue("search-admin")
		newQuery := "%" + s + "%"
		query := `SELECT name,location,description, created_at FROM
				image_blog.videos
				WHERE name LIKE ?
				`
		rows, err := db.Query(query, newQuery)
		if err != nil {
			log.Error(err.Error())
			return
		}
		var name, location, description, createdAt string
		for rows.Next() {
			err := rows.Scan(&name, &location, &description, &createdAt)
			if err != nil {
				log.Error(err.Error())
				return
			}
			image := Images{name, location, description, createdAt}
			List = append(List, image)
		}
		SentData.Statics.AdminSearch = true
		SentData.ImagesInfo = List
		tpl.ExecuteTemplate(w, "videos-admin.gohtml", &SentData)
		return
	}
	query := `SELECT name,location,description, created_at FROM
				image_blog.videos
				`
	rows, err := db.Query(query)
	if err != nil {
		log.Error(err.Error())
		return
	}
	List := []Images{}
	var name, location, description, createdAt string
	for rows.Next() {
		err := rows.Scan(&name, &location, &description, &createdAt)
		if err != nil {
			log.Error(err.Error())
			return
		}
		image := Images{name, location, description, createdAt}
		List = append(List, image)
	}
	SentData.ImagesInfo = List
	tpl.ExecuteTemplate(w, "videos-admin.gohtml", &SentData)
	return

}

func videosAdminDelete(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, _ := r.Cookie("session")
	username := strings.Split(c.Value, ",")[1]
	if !isAdmin(username) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodGet {
		r.ParseForm()
		name := r.FormValue("delete")
		if name != "" {
			db.Exec("delete from image_blog.videos where name = ?", name)
		}
		http.Redirect(w, r, "/videos-admin", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/videos-admin", http.StatusSeeOther)
	return
}

func videosAdminEdit(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, _ := r.Cookie("session")
	username := strings.Split(c.Value, ",")[1]
	if !isAdmin(username) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	SentData := &Data
	if r.Method == http.MethodGet {

		r.ParseForm()
		name := r.FormValue("name")
		if name != "" {
			query := `SELECT name,location,description, created_at FROM
				image_blog.videos where name = ?
				`
			rows, err := db.Query(query, name)
			if err != nil {
				log.Error(err.Error())
				return
			}
			List := []Images{}
			var name, location, description, createdAt string
			for rows.Next() {
				err := rows.Scan(&name, &location, &description, &createdAt)
				if err != nil {
					log.Error(err.Error())
					return
				}
				image := Images{name, location, description, createdAt}
				List = append(List, image)
			}
			SentData.ImagesInfo = List
			tpl.ExecuteTemplate(w, "edit_video_admin.gohtml", &SentData)
			return

		}
	}
	if r.Method == http.MethodPost {
		r.ParseForm()
		name := r.FormValue("name")
		location := r.FormValue("location")
		description := r.FormValue("description")
		createdAt := r.FormValue("createdAt")
		db.Exec("update image_blog.videos set location = ?, description = ?, created_at = ? where name = ?",
			location, description, createdAt, name)
		List := []Images{}
		image := Images{name, location, description, createdAt}
		List = append(List, image)
		SentData.ImagesInfo = List
		tpl.ExecuteTemplate(w, "edit_video_admin.gohtml", &SentData)
		return
	}

}

func usersAdmin(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, _ := r.Cookie("session")
	username := strings.Split(c.Value, ",")[1]
	if !isAdmin(username) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	SentData := &Data
	SentData.Admin = true
	if r.Method == http.MethodPost {
		List := []Users{}
		r.ParseForm()
		s := r.FormValue("search-admin")
		newQuery := "%" + s + "%"
		query := `SELECT username, admin FROM
				image_blog.Users
				WHERE username LIKE ?
				`
		rows, err := db.Query(query, newQuery)
		if err != nil {
			log.Error(err.Error())
			return
		}
		var userName, admin string
		for rows.Next() {
			err := rows.Scan(&userName, &admin)
			if err != nil {
				log.Error(err.Error())
				return
			}
			user := Users{userName, admin}
			List = append(List, user)
		}
		SentData.Statics.AdminSearch = true
		SentData.UsersInfo = List
		tpl.ExecuteTemplate(w, "users-admin.gohtml", &SentData)
		return
	}
	query := `SELECT username, admin FROM
				image_blog.Users
				`
	rows, err := db.Query(query)
	if err != nil {
		log.Error(err.Error())
		return
	}
	List := []Users{}
	var userName, admin string
	for rows.Next() {
		err := rows.Scan(&userName, &admin)
		if err != nil {
			log.Error(err.Error())
			return
		}
		user := Users{userName, admin}
		List = append(List, user)
	}
	SentData.UsersInfo = List
	tpl.ExecuteTemplate(w, "users-admin.gohtml", &SentData)
	return

}

func usersAdminChange(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, _ := r.Cookie("session")
	username := strings.Split(c.Value, ",")[1]
	if !isAdmin(username) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodGet {
		r.ParseForm()
		name := r.FormValue("name")
		remove := r.FormValue("remove")
		del := r.FormValue("delete")
		unBlock := r.FormValue("unblock")
		block := r.FormValue("block")
		if name != "" {
			db.Exec("update image_blog.Users set admin = 'yes' where username = ?", name)
		} else if remove != "" {
			db.Exec("update image_blog.Users set admin = 'no' where username = ?", remove)
		} else if del != "" {
			db.Exec("delete from  image_blog.Users where username = ?", del)
		} else if unBlock != "" {
			db.Exec("update image_blog.Users set retry = '0' where username = ?", unBlock)
		} else if block != "" {
			db.Exec("update image_blog.Users set retry = '6' where username = ?", block)
		}
		http.Redirect(w, r, "/users-admin", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/users-admin", http.StatusSeeOther)
	return
}

func addUserAdmin(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, _ := r.Cookie("session")
	username := strings.Split(c.Value, ",")[1]
	if !isAdmin(username) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodPost {
		r.ParseForm()
		userName := r.FormValue("username")
		password := r.FormValue("password")
		admin := r.FormValue("admin")
		if userName != "" && password != "" && admin != "" {
			encPass, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
			strPass := string(encPass)
			db.Exec("insert into image_blog.Users (username, password, admin) VALUES (?, ? ,?)", userName, strPass, admin)
		}
		http.Redirect(w, r, "/users-admin", http.StatusSeeOther)
		return
	}
	SentData := &Data

	tpl.ExecuteTemplate(w, "add-user-admin.gohtml", &SentData)
	return
}
