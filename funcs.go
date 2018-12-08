package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var parms struct {
	Username  string
	Password  string
	Ipaddress string
	Port      string
	Database  string
}

func loggedIn(w http.ResponseWriter, r *http.Request) bool {
	c, err := r.Cookie("session")
	if err != nil {
		return false
	}
	var session string
	username := strings.Split(c.Value, ",")[1]
	cookieSession := strings.Split(c.Value, ",")[0]
	dbSession := db.QueryRow("select session from image_blog.Users where username = ?", username).Scan(&session)
	if dbSession != nil {
		return false
	}
	if cookieSession != session {
		return false
	}
	db.Exec(
		`
		update image_blog.Users set last_activity = ? where username = ?
		`,
		time.Now(),
		username,
	)
	return true
}

func lastActivity() {
	log.Debug("Started")
	timeFormat := "2006-01-02 15:04:05"
	var username string
	var lastActivity string
	allUsers, _ := db.Query("select username, last_activity from image_blog.Users where session is NOT NULL")
	for allUsers.Next() {
		err := allUsers.Scan(&username, &lastActivity)
		if err != nil {
			fmt.Println(err)
			return
		}
		sessionTime, err := time.Parse(timeFormat, lastActivity)
		if err != nil {
			fmt.Println(err.Error())
		}
		timeAfterLastLogin := time.Since(sessionTime).Seconds()
		if timeAfterLastLogin > float64(cAge) {
			db.Exec("update image_blog.Users set session = NULL where username = ?", username)
		}
	}
	time.Sleep(864000000) //run every one day

}

func marchIt() string {
	f, err := os.Open("conf.json")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer f.Close()
	fb, err := ioutil.ReadAll(f)
	j := json.Unmarshal(fb, &parms)
	if j != nil {
		log.Fatal(j.Error())
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", parms.Username, parms.Password, parms.Ipaddress, parms.Port, parms.Database)
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

type SentVars struct {
	ListLength int
	PageNumber int
	Next       bool
	Prev       bool
	ListMem    []string
	ListStart  int
	ListEnd    int
	Search     string
	ImVi       []string
}

var imageSlice = 30

func pageIt(w http.ResponseWriter, s *SentVars, r *http.Request, l []string, v bool) {
	if v {
		imageSlice = 6
	} else {
		imageSlice = 30
	}
	t := len(l)
	s.ListLength = t
	r.ParseForm()
	page := r.FormValue("page")
	s.Search = r.FormValue("search")
	s.ImVi = r.Form["optradio"]
	if len(s.ImVi) == 0 {
		s.ImVi = append(s.ImVi, "image")
	}
	if strings.Contains(r.RequestURI, "page") && (!strings.HasSuffix(r.RequestURI, "page=1")) {
		s.PageNumber, err = strconv.Atoi(page)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if s.PageNumber < 0 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		s.ListStart = ((s.PageNumber - 1) * imageSlice)
		s.ListEnd = s.ListStart + imageSlice
		if !(t <= s.ListStart) {
			if t <= s.ListEnd {
				s.ListMem = l[s.ListStart:t]
				s.Next = false
			} else {
				s.ListMem = l[s.ListStart:s.ListEnd]
				s.Next = true
			}
			if s.PageNumber == 1 {
				s.Prev = false
			} else {
				s.Prev = true
			}
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return

	} else if !strings.Contains(r.RequestURI, "all") {
		s.Prev = false
		s.PageNumber = 1
		if imageSlice >= s.ListLength {
			s.Next = false
			s.ListMem = l[:s.ListLength]
		} else {
			s.Next = true
			s.ListMem = l[:imageSlice]
		}

		return
	} else {
		s.ListLength = t
		s.Next = false
		s.Prev = false
		s.PageNumber = 1
		s.ListMem = l
		return
	}
}

func updateUserSession(s, u string) error {
	_, err := db.Exec(
		`
		update image_blog.Users set session = (?) where username = (?)
		`,
		s,
		u,
	)
	return err
}

func getAndUpdateRetry(u string) (bool, error) {
	var retries string
	getRetry := db.QueryRow("select retry from image_blog.Users where username = ?", u).Scan(&retries)
	if getRetry != nil {
		return true, getRetry
	}
	setRetry, err := strconv.Atoi(retries)
	if err != nil {
		return true, err
	}
	setRetry++
	_, dbErr := db.Exec(
		`
		update image_blog.Users set retry = (?) where username = (?)
		`,
		setRetry,
		u,
	)
	if setRetry >= 5 {
		return true, dbErr
	}
	return false, dbErr

}
