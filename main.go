package main

import (
	"database/sql"
	"html/template"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

var tpl *template.Template
var db *sql.DB
var err error

func init() {
	db, err = sql.Open("mysql", "root@**********@tcp(localhost:3306)/image_blog?charset=utf8")
	if err != nil {
		log.Fatal(err)
	}
	tpl = template.Must(template.ParseGlob("templates/*.gohtml"))

}

func main() {

}
