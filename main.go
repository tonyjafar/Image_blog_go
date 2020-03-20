package main

import (
	"database/sql"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/op/go-logging"

	_ "github.com/go-sql-driver/mysql"
)

const (
	cAge           int = 86400 * 3 // stay logged in for 3 days
	widthThumbnail int = 400
)

var log = logging.MustGetLogger("appLogger.log")
var format = logging.MustStringFormatter(
	`%{time:15:04:05.000} %{shortfunc} [%{level:.4s}] %{id:03x} %{message}`,
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
	ImageDatas []ImageData
	MyVar      SentVars
	Username   string
	Statics    AdminStatics
	ImagesInfo []Images
	UsersInfo  []Users
	PassError  PassErrors
	Scharbel   ScharbelTime
}

type ImageData struct {
	Name     string
	Date     string
	Location string
}
type SearchTypes struct {
	SearchLocation string
	SearchDesc     string
	SearchDate     string
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
	t := time.Now()
	logFileName := "access-" + t.Format("2006-01-02") + ".log"
	f, err := os.OpenFile("logs/"+logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	backend2 := logging.NewLogBackend(f, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	backend1Leveled := logging.AddModuleLevel(backend2)
	backend1Leveled.SetLevel(logging.ERROR, "")
	backend1Leveled.SetLevel(logging.CRITICAL, "")
	logging.SetBackend(backend1Leveled, backend2Formatter)
	log.Debug("APP STARTED")
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
