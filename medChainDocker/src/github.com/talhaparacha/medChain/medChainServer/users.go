package main

import (
	"fmt"
	"net/http"
)

func AddUser(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/add/user")
	replyNewGenericUserRequest(w, r, "User")
}

func CommitUser(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/commit/user")
	commitNewGenericUserToChain(w, r, "User")
}
