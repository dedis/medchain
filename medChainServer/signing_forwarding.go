package main

import "net/http"

func forwardToSigning(w http.ResponseWriter, r *http.Request) {
	signingProxy.ServeHTTP(w, r)
}
