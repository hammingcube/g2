package frontend

import (
	"bytes"
	"github.com/maddyonline/problems"
	"html/template"
	"log"
	"sort"
)

func sortedList(m map[string]*problems.Problem) []*problems.Problem {
	mk := make([]string, len(m))
	i := 0
	for k, _ := range m {
		mk[i] = k
		i++
	}
	sort.Strings(mk)
	ans := []*problems.Problem{}
	for _, k := range mk {
		ans = append(ans, m[k])
	}
	return ans
}

func Index(tmpl *template.Template, probsList map[string]*problems.Problem) ([]byte, error) {
	var b bytes.Buffer
	err := tmpl.ExecuteTemplate(&b, "main", sortedList(probsList))
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return b.Bytes(), nil
}
