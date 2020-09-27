package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
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

func handelUploadVideos(chv chan *FileError, fhm *multipart.FileHeader, tn time.Time, l, d string) {
	mf, err := fhm.Open()
	if err != nil {
		log.Error(err.Error(), fhm.Filename)
		chv <- &FileError{
			IsError:   true,
			ErrorType: fmt.Sprintf("Error %v for the file %v", err.Error(), fhm.Filename),
			IsSucc:    false,
		}
		return
	}
	defer mf.Close()
	s := fhm.Size
	if !checkFileName(fhm.Filename) {
		log.Errorf("File %s name error", fhm.Filename)
		chv <- &FileError{
			IsError:   true,
			ErrorType: fmt.Sprintf("File name %v does not contian extension", fhm.Filename),
			IsSucc:    false,
		}
		return
	}
	ext := strings.Split(fhm.Filename, ".")[1]
	h := sha1.New()
	io.Copy(h, mf)
	n := fmt.Sprintf("%x", h.Sum(nil)) + "." + ext
	wd, err := os.Getwd()
	if err != nil {
		log.Error("Error %s for the file %s", err.Error(), fhm.Filename)
		chv <- &FileError{
			IsError:   true,
			ErrorType: fmt.Sprintf("Error %v for the file %v", err.Error(), fhm.Filename),
			IsSucc:    false,
		}
		return
	}
	path := filepath.Join(wd, "data/videos", n)
	nf, err := os.Create(path)
	if err != nil {
		log.Error("Error %s for the file %s", err.Error(), fhm.Filename)
		chv <- &FileError{
			IsError:   true,
			ErrorType: fmt.Sprintf("Error %v for the file %v", err.Error(), fhm.Filename),
			IsSucc:    false,
		}
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
		chv <- &FileError{
			IsError:   true,
			ErrorType: fmt.Sprintf("Error %v for the file %v", err.Error(), fhm.Filename),
			IsSucc:    false,
		}
		return
	}
	chv <- &FileError{
		IsError:   false,
		ErrorType: "",
		IsSucc:    true,
	}
}

func handelUploadImages(ch chan *FileError, fhm *multipart.FileHeader, tn time.Time, l, d string) {
	mf, err := fhm.Open()
	if err != nil {
		log.Errorf("Error: %s for the file %s", err.Error(), fhm.Filename)
		ch <- &FileError{
			IsError:   true,
			ErrorType: fmt.Sprintf("Error: %v for the file %v", err.Error(), fhm.Filename),
			IsSucc:    false,
		}
		return
	}
	defer mf.Close()
	s := fhm.Size
	if !checkFileName(fhm.Filename) {
		log.Errorf("File %s name error", fhm.Filename)
		if err != nil {
			ch <- &FileError{
				IsError:   true,
				ErrorType: fmt.Sprintf("File name %v does not contian extension", fhm.Filename),
				IsSucc:    false,
			}
			return
		}
	}
	ext := strings.Split(fhm.Filename, ".")[1]
	h := sha1.New()
	io.Copy(h, mf)
	n := fmt.Sprintf("%x", h.Sum(nil)) + "." + ext
	wd, err := os.Getwd()
	if err != nil {
		log.Errorf("Error %s for the file %s", err.Error(), fhm.Filename)
		ch <- &FileError{
			IsError:   true,
			ErrorType: fmt.Sprintf("Error %v for the file %v", err.Error(), fhm.Filename),
			IsSucc:    false,
		}
		return
	}
	path := filepath.Join(wd, "data", n)
	nf, err := os.Create(path)
	defer nf.Close()
	if err != nil {
		log.Error("Error %s for the file %s", err.Error(), fhm.Filename)
		ch <- &FileError{
			IsError:   true,
			ErrorType: fmt.Sprintf("Error %v for the file %v", err.Error(), fhm.Filename),
			IsSucc:    false,
		}
		return
	}
	mf.Seek(0, 0)
	io.Copy(nf, mf)
	image := &Image{n, l, s, tn, d}
	scrImage, err := imaging.Open("./data/" + image.Name)
	if err != nil {
		log.Error("Error %s for the file %s", err.Error(), fhm.Filename)
		mf.Close()
		nf.Close()
		os.Remove(path)
		ch <- &FileError{
			IsError:   true,
			ErrorType: fmt.Sprintf("Error %v for the file %v", err.Error(), fhm.Filename),
			IsSucc:    false,
		}
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
		ch <- &FileError{
			IsError:   true,
			ErrorType: fmt.Sprintf("Error %v for the file %v", err.Error(), fhm.Filename),
			IsSucc:    false,
		}
		return
	}
	ch <- &FileError{
		IsError:   false,
		ErrorType: "",
		IsSucc:    true,
	}
}

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
	SentData.Username = ""
	SentData.Loggedin = false
	c, err := r.Cookie("session")
	if err != nil {
		SentData.Loggedin = false
		SentData.Admin = false
		return false
	}
	var session string
	username := strings.Split(c.Value, ",")[1]
	cookieSession := strings.Split(c.Value, ",")[0]
	dbSession := db.QueryRow("select session from image_blog.Users where username = ?", username).Scan(&session)
	if dbSession != nil {
		SentData.Loggedin = false
		SentData.Admin = false
		return false
	}
	if cookieSession != session {
		SentData.Loggedin = false
		SentData.Admin = false
		getAndUpdateRetry(username)
		return false
	}
	var retries string
	getRetry := db.QueryRow("select retry from image_blog.Users where username = ?", username).Scan(&retries)
	if getRetry != nil {
		SentData.Loggedin = false
		SentData.Admin = false
		return false
	}
	setRetry, err := strconv.Atoi(retries)
	if err != nil {
		SentData.Loggedin = false
		SentData.Admin = false
		return false
	}
	if setRetry >= 5 {
		log.Criticalf("User %s is blocked", username)
		SentData.Loggedin = false
		SentData.Admin = false
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
	SentData.Username = username
	return true
}

func lastActivity() {
	log.Debug("DB Clean up Started")
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
			log.Error(err.Error())
			return
		}
		sessionTime, err := time.Parse(timeFormat, lastActivityTime)
		if err != nil {
			log.Error(err.Error())
		}
		timeAfterLastLogin := time.Since(sessionTime).Seconds()
		if timeAfterLastLogin > float64(cAge) {
			db.Exec("update image_blog.Users set session = NULL where username = ?", username)
		}
	}
	log.Debug("DB Clean up Finished")
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
	ListMem    []ImageData
	ListStart  int
	ListEnd    int
	ImVi       []string
	Slide      bool
	Searches   SearchTypes
}

var imageSlice = 30

func pageIt(w http.ResponseWriter, s *SentVars, r *http.Request, l []ImageData, v bool) {
	if v {
		imageSlice = 6
	} else {
		imageSlice = 30
	}
	t := len(l)
	s.ListLength = t
	r.ParseForm()
	page := r.FormValue("page")
	s.Searches.SearchDesc = r.FormValue("search_desc")
	s.Searches.SearchLocation = r.FormValue("search_loc")
	s.Searches.SearchDate = r.FormValue("search_date")
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
	SentData := &Data
	SentData.Admin = false
	var admin string
	getStatus := db.QueryRow("select admin from image_blog.Users where username = ?", u).Scan(&admin)
	if getStatus != nil {
		SentData.Admin = false
		return false
	}
	if admin != "yes" {
		SentData.Admin = false
		return false
	}
	SentData.Admin = true
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

func getLists() []int {
	myList := []int{}
	for len(myList) != 6 {
		rand.Seed(int64(time.Now().Nanosecond()))
		randNum := rand.Intn(50)
		if !checkList(myList, randNum) {
			myList = append(myList, randNum)
		}
	}
	return myList
}

func checkList(l []int, i int) bool {
	for _, v := range l {
		if v == i {
			return true
		}
	}
	return false
}
