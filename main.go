package main

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

const cAge int = 30

var tpl *template.Template
var db *sql.DB
var err error

func init() {
	db, err = sql.Open("mysql", "root:*************@tcp(localhost:3306)/image_blog?charset=utf8")
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
	log.Fatal(http.ListenAndServe(":8000", nil))
}
