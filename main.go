package main

import (
	"database/sql"
	"html/template"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/op/go-logging"
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
	Loggedin  bool
	Nofile    bool
	ErrorFile FileError
	UserError bool
	List      []string
	MyVar     SentVars
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
		log.Critical(err.Error())
	}
	tpl = template.Must(template.New("").Funcs(num).ParseGlob("templates/*.gohtml"))

}

func main() {
	defer db.Close()
	t := time.Now()
	logFileName := "access-" + t.Format("2006-01-02T150405") + ".log"
	f, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	backend1 := logging.NewLogBackend(f, "", 0)
	backend2 := logging.NewLogBackend(f, "", 0)
	backend2Formatter := logging.NewBackendFormatter(backend2, format)
	backend1Leveled := logging.AddModuleLevel(backend1)
	backend1Leveled.SetLevel(logging.ERROR, "")
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
	go lastActivity()
	log.Critical(http.ListenAndServe(":8000", nil))
}
