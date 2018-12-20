package main

import (
	"fmt"
	"net/http"
)

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/info/user")
	getGenericUserInfo(w, r, metaData.Users)
}

func AddUser(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/add/user")
	replyNewGenericUserRequest(w, r, "User")
}

func CommitUser(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/commit/user")
	commitNewGenericUserToChain(w, r, "User")
}
