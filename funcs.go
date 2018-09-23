package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var parms struct {
	Username  string
	Password  string
	Ipaddress string
	Port      string
	Database  string
}

func loggedIn(w http.ResponseWriter, r *http.Request) bool {
	_, err := r.Cookie("session")
	if err != nil {
		return false
	}
	return true
}

func marchIt() string {
	f, err := os.Open("conf.json")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	fb, err := ioutil.ReadAll(f)
	j := json.Unmarshal(fb, &parms)
	if j != nil {
		log.Fatal(j)
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8", parms.Username, parms.Password, parms.Ipaddress, parms.Port, parms.Database)
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
