package main

import "net/http"

// forwards the request to the signing service as a proxy
func forwardToSigning(w http.ResponseWriter, r *http.Request) {
	signingProxy.ServeHTTP(w, r)
}
