package admin_gui

import (
	"fmt"
	"html/template"
	"net/http"
)

func GUI_landing(w http.ResponseWriter, r *http.Request) {
	fmt.Println("landing")
	tmpl := template.Must(template.ParseFiles("admin_gui/templates/main_page.html"))
	tmpl.Execute(w, nil)
}
