package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"
)

func index(w http.ResponseWriter, r *http.Request) {
	SentData := &Data
	SentData.ImageDatas = nil
	c, err := r.Cookie("session")

	if err == nil {
		c.MaxAge = cAge
		http.SetCookie(w, c)
		username := strings.Split(c.Value, ",")[1]
		isAdmin(username)
	}
	if loggedIn(w, r) {
		rows, err := db.Query(
			`
			SELECT name,DATE(created_at),location FROM
			image_blog.images
			ORDER BY created_at DESC
			LIMIT 6
			`,
		)
		if err != nil {
			log.Error(err.Error())
			return
		}
		var name, date, location string
		for rows.Next() {
			err := rows.Scan(&name, &date, &location)
			if err != nil {
				log.Error(err.Error())
				return
			}
			SentData.ImageDatas = append(SentData.ImageDatas, ImageData{name, date, location})
		}
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
			isAdmin(un)
			SentData.Loggedin = true
			http.Redirect(w, r, "/images", http.StatusSeeOther)
			return
		}
		log.Errorf("Authentication Failed!! - using username %s", un)
		SentData.UserError = true
		tpl.ExecuteTemplate(w, "signin.gohtml", SentData)
		return

	}
	tpl.ExecuteTemplate(w, "signin.gohtml", SentData)
}

func images(w http.ResponseWriter, r *http.Request) {
	SentData := &Data
	SentData.ImageDatas = nil
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	username := strings.Split(c.Value, ",")[1]
	isAdmin(username)
	if strings.HasSuffix(r.RequestURI, ".css") {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	}
	c.MaxAge = cAge
	http.SetCookie(w, c)
	SentData.Loggedin = true
	rows, err := db.Query(
		`
		SELECT name,DATE(created_at),location FROM
		image_blog.images
		ORDER BY created_at DESC
		`,
	)
	if err != nil {
		log.Error(err.Error())
		return
	}
	var name, imgDate, location string
	for rows.Next() {
		err := rows.Scan(&name, &imgDate, &location)
		if err != nil {
			log.Error(err.Error())
			return
		}
		SentData.ImageDatas = append(SentData.ImageDatas, ImageData{name, imgDate, location})
	}

	pageIt(w, &SentData.MyVar, r, SentData.ImageDatas, false)
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
		ch := make(chan *FileError, len(fhs))
		for _, fhm := range fhs {
			go handelUploadImages(ch, fhm, tn, l, d)
		}
		fileErrors := []string{}
		for range fhs {
			select {
			case errors := <-ch:
				if errors.IsError {
					fileErrors = append(fileErrors, errors.ErrorType)
				}
			}

		}

		if len(fileErrors) > 0 {
			SentData.ErrorFile.IsError = true
			SentData.ErrorFile.IsSucc = false
			SentData.ErrorFile.ErrorType = strings.Join(fileErrors, " ")
		} else {
			SentData.ErrorFile.IsError = false
			SentData.ErrorFile.IsSucc = true
		}

	}

	tpl.ExecuteTemplate(w, "uplimage.gohtml", SentData)
}

func search(w http.ResponseWriter, r *http.Request) {
	SentData := &Data
	SentData.ImageDatas = nil
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	username := strings.Split(c.Value, ",")[1]
	SentData.Username = username
	if !isAdmin(username) {
		SentData.Admin = false
	} else {
		SentData.Admin = true
	}
	c.MaxAge = cAge
	var v bool
	if r.Method == http.MethodPost || strings.Contains(r.RequestURI, "page") || strings.Contains(r.RequestURI, "all") {
		video := &v
		r.ParseForm()
		sd := r.FormValue("search_desc")
		sl := r.FormValue("search_loc")
		sDate := r.FormValue("search_date")
		rad := r.Form["optradio"]
		if sl == "" && sd == "" && sDate == "" {
			tpl.ExecuteTemplate(w, "search.gohtml", SentData)
			return
		}
		if sl != "" && sd != "" && sDate != "" {
			var query string
			firstLike := "%" + sd + "%"
			secondLike := "%" + sl + "%"
			thirdLike := "%" + sDate + "%"

			if len(rad) > 0 {
				if rad[0] == "video" {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.videos
					WHERE description LIKE ?
					AND location LIKE ?
					AND created_at LIKE ?
					ORDER BY created_at DESC`
					*video = true
				} else {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE description LIKE ?
					AND location LIKE ?
					AND created_at LIKE ?
					ORDER BY created_at DESC`
					*video = false
				}
			} else {
				query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE description LIKE ?
					AND location LIKE ?
					AND created_at LIKE ?
					ORDER BY created_at DESC`
				*video = false
			}
			rows, err := db.Query(query, firstLike, secondLike, thirdLike)
			if err != nil {
				log.Error(err.Error())
				return
			}
			var name, date, location string
			for rows.Next() {
				err := rows.Scan(&name, &date, &location)
				if err != nil {
					log.Error(err.Error())
					return
				}
				SentData.ImageDatas = append(SentData.ImageDatas, ImageData{name, date, location})
			}

		} else if sl != "" && sd != "" {
			var query string
			firstLike := "%" + sd + "%"
			secondLike := "%" + sl + "%"

			if len(rad) > 0 {
				if rad[0] == "video" {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.videos
					WHERE description LIKE ?
					AND location LIKE ?
					ORDER BY created_at DESC`
					*video = true
				} else {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE description LIKE ?
					AND location LIKE ?
					ORDER BY created_at DESC`
					*video = false
				}
			} else {
				query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE description LIKE ?
					AND location LIKE ?
					ORDER BY created_at DESC`
				*video = false
			}
			rows, err := db.Query(query, firstLike, secondLike)
			if err != nil {
				log.Error(err.Error())
				return
			}
			var name, date, location string
			for rows.Next() {
				err := rows.Scan(&name, &date, &location)
				if err != nil {
					log.Error(err.Error())
					return
				}
				SentData.ImageDatas = append(SentData.ImageDatas, ImageData{name, date, location})
			}

		} else if sDate != "" && sd != "" {
			var query string
			firstLike := "%" + sd + "%"
			secondLike := "%" + sDate + "%"

			if len(rad) > 0 {
				if rad[0] == "video" {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.videos
					WHERE description LIKE ?
					AND created_at LIKE ?
					ORDER BY created_at DESC`
					*video = true
				} else {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE description LIKE ?
					AND created_at LIKE ?
					ORDER BY created_at DESC`
					*video = false
				}
			} else {
				query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE description LIKE ?
					AND created_at LIKE ?
					ORDER BY created_at DESC`
				*video = false
			}
			rows, err := db.Query(query, firstLike, secondLike)
			if err != nil {
				log.Error(err.Error())
				return
			}
			var name, date, location string
			for rows.Next() {
				err := rows.Scan(&name, &date, &location)
				if err != nil {
					log.Error(err.Error())
					return
				}
				SentData.ImageDatas = append(SentData.ImageDatas, ImageData{name, date, location})
			}

		} else if sDate != "" && sl != "" {
			var query string
			firstLike := "%" + sl + "%"
			secondLike := "%" + sDate + "%"

			if len(rad) > 0 {
				if rad[0] == "video" {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.videos
					WHERE location LIKE ?
					AND created_at LIKE ?
					ORDER BY created_at DESC`
					*video = true
				} else {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE location LIKE ?
					AND created_at LIKE ?
					ORDER BY created_at DESC`
					*video = false
				}
			} else {
				query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE location LIKE ?
					AND created_at LIKE ?
					ORDER BY created_at DESC`
				*video = false
			}
			rows, err := db.Query(query, firstLike, secondLike)
			if err != nil {
				log.Error(err.Error())
				return
			}
			var name, date, location string
			for rows.Next() {
				err := rows.Scan(&name, &date, &location)
				if err != nil {
					log.Error(err.Error())
					return
				}
				SentData.ImageDatas = append(SentData.ImageDatas, ImageData{name, date, location})
			}

		} else if sd != "" {
			var query string
			firstLike := "%" + sd + "%"

			if len(rad) > 0 {
				if rad[0] == "video" {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.videos
					WHERE description LIKE ?
					ORDER BY created_at DESC`
					*video = true
				} else {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE description LIKE ?
					ORDER BY created_at DESC`
					*video = false
				}
			} else {
				query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE description LIKE ?
					ORDER BY created_at DESC`
				*video = false
			}
			rows, err := db.Query(query, firstLike)
			if err != nil {
				log.Error(err.Error())
				return
			}
			var name, date, location string
			for rows.Next() {
				err := rows.Scan(&name, &date, &location)
				if err != nil {
					log.Error(err.Error())
					return
				}
				SentData.ImageDatas = append(SentData.ImageDatas, ImageData{name, date, location})
			}

		} else if sl != "" {
			var query string
			firstLike := "%" + sl + "%"

			if len(rad) > 0 {
				if rad[0] == "video" {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.videos
					WHERE location LIKE ?
					ORDER BY created_at DESC`
					*video = true
				} else {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE location LIKE ?
					ORDER BY created_at DESC`
					*video = false
				}
			} else {
				query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE location LIKE ?
					ORDER BY created_at DESC`
				*video = false
			}
			rows, err := db.Query(query, firstLike)
			if err != nil {
				log.Error(err.Error())
				return
			}
			var name, date, location string
			for rows.Next() {
				err := rows.Scan(&name, &date, &location)
				if err != nil {
					log.Error(err.Error())
					return
				}
				SentData.ImageDatas = append(SentData.ImageDatas, ImageData{name, date, location})
			}

		} else if sDate != "" {
			var query string
			firstLike := "%" + sDate + "%"

			if len(rad) > 0 {
				if rad[0] == "video" {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.videos
					WHERE created_at LIKE ?
					ORDER BY created_at DESC`
					*video = true
				} else {
					query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE created_at LIKE ?
					ORDER BY created_at DESC`
					*video = false
				}
			} else {
				query = `SELECT name,DATE(created_at),location FROM
					image_blog.images
					WHERE created_at LIKE ?
					ORDER BY created_at DESC`
				*video = false
			}
			rows, err := db.Query(query, firstLike)
			if err != nil {
				log.Error(err.Error())
				return
			}
			var name, date, location string
			for rows.Next() {
				err := rows.Scan(&name, &date, &location)
				if err != nil {
					log.Error(err.Error())
					return
				}
				SentData.ImageDatas = append(SentData.ImageDatas, ImageData{name, date, location})
			}

		}

	}
	pageIt(w, &SentData.MyVar, r, SentData.ImageDatas, v)
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
		chv := make(chan *FileError, len(fhs))
		for _, fhm := range fhs {
			go handelUploadVideos(chv, fhm, tn, l, d)
		}
		fileErrors := []string{}
		for range fhs {
			select {
			case errors := <-chv:
				if errors.IsError {
					fileErrors = append(fileErrors, errors.ErrorType)
				}
			}

		}
		if len(fileErrors) > 0 {
			SentData.ErrorFile.IsError = true
			SentData.ErrorFile.IsSucc = false
			SentData.ErrorFile.ErrorType = strings.Join(fileErrors, " ")
		} else {
			SentData.ErrorFile.IsError = false
			SentData.ErrorFile.IsSucc = true
		}
	}
	tpl.ExecuteTemplate(w, "uploadvideo.gohtml", SentData)
}

func videos(w http.ResponseWriter, r *http.Request) {
	SentData := &Data
	SentData.ImageDatas = nil
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, err := r.Cookie("session")
	if err != nil {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c.MaxAge = cAge
	http.SetCookie(w, c)
	username := strings.Split(c.Value, ",")[1]
	isAdmin(username)
	rows, err := db.Query(
		`
		SELECT name,DATE(created_at),location FROM
		image_blog.videos
		ORDER BY created_at DESC
		`,
	)
	if err != nil {
		log.Error(err.Error())
		return
	}
	var name, date, location string
	for rows.Next() {
		err := rows.Scan(&name, &date, &location)
		if err != nil {
			log.Error(err.Error())
			return
		}
		SentData.ImageDatas = append(SentData.ImageDatas, ImageData{name, date, location})
	}
	pageIt(w, &SentData.MyVar, r, SentData.ImageDatas, true)
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
	var imageCount, videoCount, userCount, blockedUser, imageMonthVar, imageCountVar, imageYearVar, imageYearCountVar,
		videoMonthVar, videoYearVar, videoCountVar, videoCounYeartVar, imagesSize, videosSize, sizeDB,
		imageDesc, imageDescCount, imageLoc, imageLocCount, videoDesc, videoDescCount, videoLoc, videoLocCount string
	SentData.Statics.ImagesByMonths = nil
	SentData.Statics.ImagesByYears = nil
	SentData.Statics.VideosByMonths = nil
	SentData.Statics.VideosByYears = nil
	SentData.Statics.VideosDesc = nil
	SentData.Statics.VideosLoc = nil
	SentData.Statics.ImagesDesc = nil
	SentData.Statics.ImagesLoc = nil
	db.QueryRow("select count(*) from image_blog.images").Scan(&imageCount)
	db.QueryRow("select count(*) from image_blog.videos").Scan(&videoCount)
	db.QueryRow("select count(*) from image_blog.Users").Scan(&userCount)
	db.QueryRow("select count(*) from image_blog.Users where retry >= 5").Scan(&blockedUser)
	db.QueryRow("select format(sum(size)/1024/1024/1024, 2) from image_blog.images").Scan(&imagesSize)
	db.QueryRow("select format(sum(size)/1024/1024/1024,2) from image_blog.videos").Scan(&videosSize)
	db.QueryRow("SELECT format(sum( data_length + index_length ) / 1024 / 1024/ 1024, 2) \"database size in GB\" FROM information_schema.TABLES WHERE table_schema = \"image_blog\"").Scan(&sizeDB)
	getImageByMonth, _ := db.Query("select monthname(created_at), count(*) from images group by monthname(created_at)")
	getImageByYear, _ := db.Query("select year(created_at), count(*) from images group by year(created_at)")
	getVideoByMonth, _ := db.Query("select monthname(created_at), count(*) from videos group by monthname(created_at)")
	getVideoByYear, _ := db.Query("select year(created_at), count(*) from videos group by year(created_at)")
	getImageDesc, _ := db.Query("select description, count(*) from images group by description")
	getImageLoc, _ := db.Query("select location, count(*) from images group by location")
	getVideoDesc, _ := db.Query("select description, count(*) from videos group by description")
	getVideoLoc, _ := db.Query("select location, count(*) from videos group by location")

	for getImageDesc.Next() {
		getImageDesc.Scan(&imageDesc, &imageDescCount)
		SentData.Statics.ImagesDesc = append(SentData.Statics.ImagesDesc, ImageDesc{imageDesc, imageDescCount})
	}

	for getImageLoc.Next() {
		getImageLoc.Scan(&imageLoc, &imageLocCount)
		SentData.Statics.ImagesLoc = append(SentData.Statics.ImagesLoc, ImageLoc{imageLoc, imageLocCount})
	}

	for getVideoDesc.Next() {
		getVideoDesc.Scan(&videoDesc, &videoDescCount)
		SentData.Statics.VideosDesc = append(SentData.Statics.VideosDesc, VideoDesc{videoDesc, videoDescCount})
	}

	for getVideoLoc.Next() {
		getVideoLoc.Scan(&videoLoc, &videoLocCount)
		SentData.Statics.VideosLoc = append(SentData.Statics.VideosLoc, VideoLoc{videoLoc, videoLocCount})
	}

	for getImageByMonth.Next() {
		getImageByMonth.Scan(&imageMonthVar, &imageCountVar)
		SentData.Statics.ImagesByMonths = append(SentData.Statics.ImagesByMonths, ImageByMonth{imageMonthVar, imageCountVar})
	}
	for getImageByYear.Next() {
		getImageByYear.Scan(&imageYearVar, &imageYearCountVar)
		SentData.Statics.ImagesByYears = append(SentData.Statics.ImagesByYears, ImageByYear{imageYearVar, imageYearCountVar})
	}
	for getVideoByMonth.Next() {
		getVideoByMonth.Scan(&videoMonthVar, &videoCountVar)
		SentData.Statics.VideosByMonths = append(SentData.Statics.VideosByMonths, VideoByMonth{videoMonthVar, videoCountVar})
	}
	for getVideoByYear.Next() {
		getVideoByYear.Scan(&videoYearVar, &videoCounYeartVar)
		SentData.Statics.VideosByYears = append(SentData.Statics.VideosByYears, VideoByYear{videoYearVar, videoCounYeartVar})
	}
	SentData.Statics.SizeDB = sizeDB
	SentData.Statics.VideosSize = videosSize
	SentData.Statics.ImageSize = imagesSize
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
				image_blog.images ORDER BY created_at DESC
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
				image_blog.videos ORDER BY created_at DESC
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
				image_blog.videos where name = ? ORDER BY created_at DESC
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
	SentData := &Data
	SentData.PassError.IsError = false
	SentData.PassError.IsSucc = false
	if r.Method == http.MethodPost {
		r.ParseForm()
		userName := r.FormValue("username")
		password := r.FormValue("password")
		admin := r.FormValue("admin")
		if userName != "" && password != "" && admin != "" {
			check, err := passPolicy(userName, password)
			if !check {
				SentData.PassError.IsError = true
				SentData.PassError.IsSucc = false
				SentData.PassError.ErrorType = err
				tpl.ExecuteTemplate(w, "add-user-admin.gohtml", &SentData)
				return
			}
			encPass, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
			strPass := string(encPass)
			db.Exec("insert into image_blog.Users (username, password, admin) VALUES (?, ? ,?)", userName, strPass, admin)
		} else {
			SentData.PassError.IsError = true
			SentData.PassError.IsSucc = false
			SentData.PassError.ErrorType = "Please fill on all infos"
			tpl.ExecuteTemplate(w, "add-user-admin.gohtml", &SentData)
			return
		}
		SentData.PassError.IsError = false
		SentData.PassError.IsSucc = true
		http.Redirect(w, r, "/users-admin", http.StatusSeeOther)
		return
	}

	tpl.ExecuteTemplate(w, "add-user-admin.gohtml", &SentData)
	return
}

func getInfo(w http.ResponseWriter, r *http.Request) {
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
	ImageInfos = []ImageInfo{}
	rows, err := db.Query(
		`
		SELECT name,location,description,size,created_at FROM
		image_blog.images
		ORDER BY created_at DESC
		`,
	)
	if err != nil {
		log.Error(err.Error())
		return
	}
	var name, location, description, size, created_at string
	for rows.Next() {
		err := rows.Scan(&name, &location, &description, &size, &created_at)
		if err != nil {
			log.Error(err.Error())
			return
		}
		ImageInfos = append(ImageInfos, ImageInfo{name, location, description, size, created_at})
	}
	result := executeQuery(w, r.URL.Query().Get("query"), schema)
	json.NewEncoder(w).Encode(result)
}

func getScharbelTime(w http.ResponseWriter, r *http.Request) {
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
	timeFormat := "2006-01-02 15:04:05 CET"
	scharbelTime := "2019-03-05 09:50:00 CET"
	loc, _ := time.LoadLocation("Europe/Berlin")
	parseTime, _ := time.ParseInLocation(timeFormat, scharbelTime, loc)
	getSeconds := int(time.Since(parseTime).Seconds())
	SentData.Scharbel.Years = getSeconds / 31557600
	SentData.Scharbel.Months = (getSeconds % 31557600) / 2592000
	SentData.Scharbel.Days = (getSeconds % 2592000) / 86400
	SentData.Scharbel.Hours = (getSeconds % 86400) / 3600
	SentData.Scharbel.Minutes = (getSeconds % 3600) / 60
	SentData.Scharbel.Seconds = (getSeconds % 3600) % 60
	if r.Method == http.MethodPost {
		tpl.ExecuteTemplate(w, "scharbel.gohtml", &SentData)
		return
	}
	tpl.ExecuteTemplate(w, "scharbel.gohtml", &SentData)
	return

}

func playIt(w http.ResponseWriter, r *http.Request) {
	if !loggedIn(w, r) {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	c, _ := r.Cookie("session")
	username := strings.Split(c.Value, ",")[1]
	isAdmin(username)
	PlayData := &SechsAus49{
		Lists:       [][]int{},
		SuperNumber: 0,
		LosNumber:   "",
	}
	for len(PlayData.Lists) != 14 {
		newList := getLists()
		PlayData.Lists = append(PlayData.Lists, newList)
	}

	rand.Seed(int64(time.Now().Nanosecond()))
	PlayData.SuperNumber = rand.Intn(10)
	losNum := []int{}
	for len(losNum) != 6 {
		rand.Seed(int64(time.Now().Nanosecond()))
		randNum := rand.Intn(10)
		losNum = append(losNum, randNum)
	}
	losNnumStr := []string{}
	for _, v := range losNum {
		losNnumStr = append(losNnumStr, strconv.Itoa(v))
	}
	PlayData.LosNumber = strings.Join(losNnumStr, "")
	tpl.ExecuteTemplate(w, "6aus49.gohtml", &PlayData)
	return

}
