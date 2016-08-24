package main

import (
	"github.com/maddyonline/problems"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func main() {
	//{{range .Items}}
	dirname, err := filepath.Abs("../../problems")
	if err != nil {
		log.Fatal(err)
		return
	}
	problems, err := problems.GetList(dirname, ioutil.Discard)
	if err != nil {
		log.Fatal(err)
		return
	}
	tmpl := template.Must(template.ParseFiles("templates/problems_list.tpl", "templates/main.tpl"))
	err = tmpl.ExecuteTemplate(os.Stdout, "main", problems["problems"])
	if err != nil {
		log.Fatal(err)
		return
	}
}
