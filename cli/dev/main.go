package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/dcb9/testOAuth/proxy"
	"github.com/rs/cors"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", welcomeHandler)
	mux.HandleFunc("/oauth/twitter/authorize_url", twitterAuthorizeURLHandler)
	mux.HandleFunc("/oauth/twitter/callback", twitterCallbackHandler)
	handler := cors.Default().Handler(mux)

	err := http.ListenAndServe("localhost:1235", handler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func twitterAuthorizeURLHandler(w http.ResponseWriter, req *http.Request) {
	loginURL := proxy.HandleTwitterLoginURL()
	fmt.Fprint(w, loginURL)
}

func twitterCallbackHandler(w http.ResponseWriter, req *http.Request) {
	userBytes, err := proxy.HandleTwitterCallback(req)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	fmt.Fprint(w, string(userBytes))
}

// welcomeHandler shows a welcome message and login button.
func welcomeHandler(w http.ResponseWriter, req *http.Request) {
	proxy.RenderIndexHTML(proxy.IndexHTMLData{
		TwitterAuthorizeURLApi: template.URL("/oauth/twitter/authorize_url"),
		TwitterCallbackURL:     template.URL("/oauth/twitter/callback"),
	}, w)
}
