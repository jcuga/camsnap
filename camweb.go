package main

import (
	"crypto/subtle"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
)

func main() {
	frameFile := flag.String("frame", "./camera_frame.jpeg", "Location of camera frame")
	listenAddr := flag.String("addr", ":9876", "Webserver listen address")
	flag.Parse()
	http.HandleFunc("/", BasicAuth(getHandleIndex(*frameFile), "bubby", "iseeu2bigguy", "Please enter your username and password for this site"))
	log.Printf("Serving camera frame file: %v via http on: %v", *frameFile, *listenAddr)
	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		log.Fatalf("Error setting up webserver: %v", err)
		os.Exit(1)
	}
}

func getHandleIndex(frameFile string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		frameData, err := ioutil.ReadFile(frameFile)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error reading frame: %v", err)))
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", strconv.Itoa(len(frameData)))
		if _, err := w.Write(frameData); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("Error writing frame image to response.  error: %v", err)))
			return
		}
	}
}

// basic auth from:  http://stackoverflow.com/questions/21936332/idiomatic-way-of-requiring-http-basic-auth-in-go/39591234#39591234

// BasicAuth wraps a handler requiring HTTP basic auth for it using the given
// username and password and the specified realm, which shouldn't contain quotes.
//
// Most web browser display a dialog with something like:
//
//    The website says: "<realm>"
//
// Which is really stupid so you may want to set the realm to a message rather than
// an actual realm.
func BasicAuth(handler http.HandlerFunc, username, password, realm string) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		user, pass, ok := r.BasicAuth()

		if !ok || subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 || subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			w.WriteHeader(401)
			w.Write([]byte("Unauthorised.\n"))
			return
		}

		handler(w, r)
	}
}
