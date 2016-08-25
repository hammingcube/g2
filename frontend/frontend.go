package frontend

import (
	"bytes"
	"github.com/maddyonline/problems"
	"html/template"
	"io/ioutil"
	"log"
	"path/filepath"
)

func Index(rootdir string) ([]byte, error) {
	//{{range .Items}}
	dirname, err := filepath.Abs(filepath.Join(rootdir, "../../problems"))
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	problems, err := problems.GetList(dirname, ioutil.Discard)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	tmpl := template.Must(template.ParseFiles(
		filepath.Join(rootdir, "templates/problems_list.tpl"),
		filepath.Join(rootdir, "templates/main.tpl"),
	))
	var b bytes.Buffer
	err = tmpl.ExecuteTemplate(&b, "main", problems["problems"])
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return b.Bytes(), nil
}
