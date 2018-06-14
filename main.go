package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

const (
	cAge int = 86400 * 3 // stay logged in for 3 days
)

var tpl *template.Template
var db *sql.DB
var err error

var data struct {
	loggedin  bool
	nofile    bool
	fileerror bool
}

func init() {
	db, err = sql.Open("mysql", marchIt())
	if err != nil {
		log.Fatal(err)
	}
	tpl = template.Must(template.ParseGlob("templates/*.gohtml"))

}

func main() {
	defer db.Close()
	http.Handle("/favicon.ico", http.NotFoundHandler())
	http.HandleFunc("/", index)
	http.HandleFunc("/signin", login)
	http.HandleFunc("/images", images)
	http.HandleFunc("/signout", signout)
	http.HandleFunc("/add_image", addImage)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
