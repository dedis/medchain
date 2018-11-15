package main

import (
	"fmt"
	"html/template"
	"net/http"
)

func adminLanding(w http.ResponseWriter, r *http.Request) {
	fmt.Println("adminlanding")
	tmpl := template.Must(template.ParseFiles("templates/static/admin_landing.html"))
	tmpl.Execute(w, nil)
}
