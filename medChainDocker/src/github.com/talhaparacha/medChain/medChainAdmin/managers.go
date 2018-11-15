package main

import (
	"fmt"
	"html/template"
	"net/http"
)

func managerLanding(w http.ResponseWriter, r *http.Request) {
	fmt.Println("managerlanding")
	tmpl := template.Must(template.ParseFiles("templates/static/manager_landing.html"))
	tmpl.Execute(w, nil)
}
