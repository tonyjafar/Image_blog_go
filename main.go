package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

const (
	cAge           int = 86400 * 3 // stay logged in for 3 days
	widthThumbnail int = 400
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
		log.Fatal(err)
	}
	tpl = template.Must(template.New("").Funcs(num).ParseGlob("templates/*.gohtml"))

}

func main() {
	defer db.Close()
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
	log.Fatal(http.ListenAndServe(":8000", nil))
}
