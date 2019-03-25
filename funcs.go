package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/graphql-go/graphql"
)

var parms struct {
	Username  string
	Password  string
	Ipaddress string
	Port      string
	Database  string
}

type ImageInfo struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Description string `json:"description"`
	Size        string `json:"size"`
	Created_at  string `json:"created_at"`
}

var ImageInfos []ImageInfo

var imageType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Image",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Type: graphql.String,
			},
			"location": &graphql.Field{
				Type: graphql.String,
			},
			"description": &graphql.Field{
				Type: graphql.String,
			},
			"size": &graphql.Field{
				Type: graphql.Int,
			},
			"created_at": &graphql.Field{
				Type: graphql.String,
			},
		},
	},
)

var queryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Query",
		Fields: graphql.Fields{
			/* Get (read) single image by name
			   http://localhost:8000/info?query={image(name:"name"){name,description,created_at}}
			*/
			"image": &graphql.Field{
				Type:        imageType,
				Description: "Get image by name",
				Args: graphql.FieldConfigArgument{
					"name": &graphql.ArgumentConfig{
						Type: graphql.String,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					name, ok := p.Args["name"].(string)
					if ok {
						// Find product
						for _, image := range ImageInfos {
							if string(image.Name) == name {
								return image, nil
							}
						}
					}
					return nil, nil
				},
			},
			/* Get (read) image list
			   http://localhost:8000/info?query={list{name,description,created_at}}
			*/
			"list": &graphql.Field{
				Type:        graphql.NewList(imageType),
				Description: "Get images list",
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					return ImageInfos, nil
				},
			},
		},
	})

var mutationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Mutation",
	Fields: graphql.Fields{
		/* Delete Image by name
		   http://localhost:8080/info?query=mutation+_{delete(name:"name"){name,description,created_at}}
		*/
		"delete": &graphql.Field{
			Type:        imageType,
			Description: "Delete image by name",
			Args: graphql.FieldConfigArgument{
				"name": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(graphql.String),
				},
			},
			Resolve: func(params graphql.ResolveParams) (interface{}, error) {
				name, _ := params.Args["name"].(string)
				image := ImageInfo{}
				for i, p := range ImageInfos {
					if name == p.Name {
						image = ImageInfos[i]
						// Remove from DB
						db.Exec("delete from image_blog.images where name = ?", name)
					}
				}

				return image, nil
			},
		},
	},
})

var schema, _ = graphql.NewSchema(
	graphql.SchemaConfig{
		Query:    queryType,
		Mutation: mutationType,
	},
)

func executeQuery(w http.ResponseWriter, query string, schema graphql.Schema) *graphql.Result {
	result := graphql.Do(graphql.Params{
		Schema:        schema,
		RequestString: query,
	})
	if len(result.Errors) > 0 {
		log.Errorf("errors: %v", result.Errors)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Bad Request!\n"))
	}
	return result
}

func loggedIn(w http.ResponseWriter, r *http.Request) bool {
	SentData := &Data
	c, err := r.Cookie("session")
	if err != nil {
		SentData.Loggedin = false
		return false
	}
	var session string
	username := strings.Split(c.Value, ",")[1]
	cookieSession := strings.Split(c.Value, ",")[0]
	dbSession := db.QueryRow("select session from image_blog.Users where username = ?", username).Scan(&session)
	if dbSession != nil {
		SentData.Loggedin = false
		return false
	}
	if cookieSession != session {
		SentData.Loggedin = false
		getAndUpdateRetry(username)
		return false
	}
	var retries string
	getRetry := db.QueryRow("select retry from image_blog.Users where username = ?", username).Scan(&retries)
	if getRetry != nil {
		return false
	}
	setRetry, err := strconv.Atoi(retries)
	if err != nil {
		return false
	}
	if setRetry >= 5 {
		log.Criticalf("User %s is blocked", username)
		return false
	}

	db.Exec(
		`
		update image_blog.Users set last_activity = ? where username = ?
		`,
		time.Now(),
		username,
	)
	SentData.Loggedin = true
	return true
}

func lastActivity() {
	log.Debug("Started")
	timeFormat := "2006-01-02 15:04:05"
	var username string
	var lastActivityTime string
	allUsers, err := db.Query("select username, last_activity from image_blog.Users where session is NOT NULL")
	if err != nil {
		log.Error(err.Error())
	}
	for allUsers.Next() {
		err := allUsers.Scan(&username, &lastActivityTime)
		if err != nil {
			log.Fatal(err.Error())
			return
		}
		sessionTime, err := time.Parse(timeFormat, lastActivityTime)
		if err != nil {
			fmt.Println(err.Error())
		}
		timeAfterLastLogin := time.Since(sessionTime).Seconds()
		if timeAfterLastLogin > float64(cAge) {
			db.Exec("update image_blog.Users set session = NULL where username = ?", username)
		}
	}
	log.Debug("Finished")
	time.Sleep(24 * time.Hour) //run every one day
	lastActivity()
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
	Slide      bool
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
	slide := r.FormValue("slide")
	if len(s.ImVi) == 0 {
		s.ImVi = append(s.ImVi, "image")
	}
	if slide == "true" {
		s.Slide = true
	} else {
		s.Slide = false
	}
	if strings.Contains(r.RequestURI, "page") && (!strings.HasSuffix(r.RequestURI, "page=1")) {
		s.PageNumber, err = strconv.Atoi(page)
		if err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if s.PageNumber <= 0 {
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
		log.Criticalf("User %s is blocked", u)
		return true, dbErr
	}
	return false, dbErr

}

func checkFileName(f string) bool {
	match, _ := regexp.MatchString("^.+\\.\\w{1,}", f)
	return match
}

func isAdmin(u string) bool {
	var admin string
	getStatus := db.QueryRow("select admin from image_blog.Users where username = ?", u).Scan(&admin)
	if getStatus != nil {
		return false
	}
	if admin != "yes" {
		return false
	}
	return true
}

func passPolicy(u, p string) (bool, string) {
	var newUsername string
	db.QueryRow("select username from image_blog.Users where username = ?", u).Scan(&newUsername)
	if newUsername != "" {
		return false, "Username is already taken"
	}
	if len(p) < 6 {
		return false, "Password must be at least 6 charachters long"
	}
	checkSmallLetters, _ := regexp.MatchString("[a-z]", p)
	if !checkSmallLetters {
		return checkSmallLetters, "Password must contain at least one small letter"
	}
	checkCapLetters, _ := regexp.MatchString("[A-Z]", p)
	if !checkCapLetters {
		return checkCapLetters, "Password must contain at least one capital letter"
	}
	checkNonChar, _ := regexp.MatchString("\\W", p)
	if !checkNonChar {
		return checkNonChar, "Password must contain at least one special letter"
	}
	checkNum, _ := regexp.MatchString("[0-9]", p)
	if !checkNum {
		return checkNum, "Password must contain at least one number"
	}
	return true, "Sucess"
}
