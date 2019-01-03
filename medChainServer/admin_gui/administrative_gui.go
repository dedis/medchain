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

func UserInfoPage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("user_info")
	tmpl := template.Must(template.ParseFiles("admin_gui/templates/info_page/generic_user_info.html"))
	tmpl.Execute(w, nil)
}

func ProjectInfoPage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("project_info")
	tmpl := template.Must(template.ParseFiles("admin_gui/templates/info_page/project_info.html"))
	tmpl.Execute(w, nil)
}

func HospitalInfoPage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("hospital_info")
	tmpl := template.Must(template.ParseFiles("admin_gui/templates/info_page/hospital_info.html"))
	tmpl.Execute(w, nil)
}

func ActionInfoPage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("action_info")
	tmpl := template.Must(template.ParseFiles("admin_gui/templates/info_page/action_info.html"))
	tmpl.Execute(w, nil)
}

func DarcInfoPage(w http.ResponseWriter, r *http.Request) {
	fmt.Println("darc_info")
	tmpl := template.Must(template.ParseFiles("admin_gui/templates/info_page/darc_info.html"))
	tmpl.Execute(w, nil)
}
