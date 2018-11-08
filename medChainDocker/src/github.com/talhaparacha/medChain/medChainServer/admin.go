package main

import (
	"net/http"
)

func newManager(w http.ResponseWriter, r *http.Request) {

}

func newAdministrator(w http.ResponseWriter, r *http.Request) {
	sayHello(w, r)
}
