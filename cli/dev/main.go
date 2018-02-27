package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/dcb9/testOAuth/proxy"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", welcomeHandler)
	mux.HandleFunc("/twitter/login-url", twitterLoginURLHandler)
	mux.HandleFunc("/twitter/user-info", twitterUserInfoHandler)
	err := http.ListenAndServe("localhost:1234", mux)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func twitterLoginURLHandler(w http.ResponseWriter, req *http.Request) {
	loginURL := proxy.HandleTwitterLoginURL()
	fmt.Fprint(w, loginURL)
}

func twitterUserInfoHandler(w http.ResponseWriter, req *http.Request) {
	userBytes, err := proxy.HandleTwitterUserInfo(req)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Fprint(w, string(userBytes))
}

// welcomeHandler shows a welcome message and login button.
func welcomeHandler(w http.ResponseWriter, req *http.Request) {
	proxy.RenderIndexHTML(proxy.IndexHTMLData{
		GetTwitterAuthorizationURLApi: template.URL("/twitter/login-url"),
		GetTwitterUserInfoApi:         template.URL("/twitter/user-info"),
	}, w)
}
