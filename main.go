package main

import (
	"database/sql"
	"html/template"
	"net/http"

	"log"

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	cAge           int = 86400 * 3 // stay logged in for 3 days
	widthThumbnail int = 400
)

var tpl *template.Template
var db *sql.DB
var err error

var Data struct {
	Loggedin   bool
	Admin      bool
	Nofile     bool
	ErrorFile  FileError
	UserError  bool
	List       []string
	MyVar      SentVars
	Username   string
	Statics    AdminStatics
	ImagesInfo []Images
	UsersInfo  []Users
	PassError  PassErrors
	Scharbel   ScharbelTime
}

type ScharbelTime struct {
	Years   int
	Months  int
	Days    int
	Hours   int
	Minutes int
	Seconds int
}

type PassErrors struct {
	IsError   bool
	ErrorType string
	IsSucc    bool
}

type Images struct {
	Name        string
	Location    string
	Description string
	CreatedAt   string
}

type Users struct {
	Username string
	Admin    string
}

type AdminStatics struct {
	ImageCount     string
	VideoCount     string
	UserCount      string
	BlockedUser    string
	AdminSearch    bool
	ImageSize      string
	VideosSize     string
	SizeDB         string
	ImagesByMonths []ImageByMonth
	ImagesByYears  []ImageByYear
	VideosByMonths []VideoByMonth
	VideosByYears  []VideoByYear
	ImagesDesc     []ImageDesc
	ImagesLoc      []ImageLoc
	VideosDesc     []VideoDesc
	VideosLoc      []VideoLoc
}

type ImageDesc struct {
	Desc  string
	Count string
}

type ImageLoc struct {
	Loc   string
	Count string
}

type VideoDesc struct {
	Desc  string
	Count string
}

type VideoLoc struct {
	Loc   string
	Count string
}

type ImageByMonth struct {
	Month string
	Count string
}

type VideoByMonth struct {
	Month string
	Count string
}

type ImageByYear struct {
	Year  string
	Count string
}

type VideoByYear struct {
	Year  string
	Count string
}

type FileError struct {
	IsError   bool
	ErrorType string
	IsSucc    bool
}

var num = template.FuncMap{
	"add": add,
	"red": red,
}

var l = &lumberjack.Logger{
	Filename:   "logs/APP.log",
	MaxSize:    500,
	MaxBackups: 10,
	MaxAge:     1,
	Compress:   true,
}

func add(p int) int {
	return p + 1
}

func red(p int) int {
	return p - 1
}

func init() {
	db, err = sql.Open("mysql", marchIt())
	if err != nil {
		log.Fatal(err.Error())
	}
	tpl = template.Must(template.New("").Funcs(num).ParseGlob("templates/*.gohtml"))

}

func main() {
	defer db.Close()
	log.SetOutput(l)
	log.Println("APP Started")
	http.HandleFunc("/assets/", handleFileServer("./data", "/assets"))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.HandleFunc("/", index)
	http.HandleFunc("/signin", login)
	http.HandleFunc("/images", images)
	http.HandleFunc("/videos", videos)
	http.HandleFunc("/signout", signout)
	http.HandleFunc("/add_image", addImage)
	http.HandleFunc("/add_video", addVideo)
	http.HandleFunc("/search", search)
	http.HandleFunc("/admin", admin)
	http.HandleFunc("/images-admin", imagesAdmin)
	http.HandleFunc("/delete-image", imagesAdminDelete)
	http.HandleFunc("/edit-image", imagesAdminEdit)
	http.HandleFunc("/videos-admin", videosAdmin)
	http.HandleFunc("/delete-video", videosAdminDelete)
	http.HandleFunc("/edit-video", videosAdminEdit)
	http.HandleFunc("/users-admin", usersAdmin)
	http.HandleFunc("/edit-user", usersAdminChange)
	http.HandleFunc("/add-user", addUserAdmin)
	http.HandleFunc("/info", getInfo)
	http.HandleFunc("/scharbel", getScharbelTime)
	go lastActivity()
	log.Fatal(http.ListenAndServe(":8000", nil))
}
