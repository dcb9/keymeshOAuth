package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dcb9/keymeshOAuth/proxy"
	"github.com/rs/cors"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/twitter/authorize_url", twitterAuthorizeURLHandler)
	mux.HandleFunc("/oauth/twitter/callback", twitterCallbackHandler)
	mux.HandleFunc("/oauth/twitter/verify", twitterVerifyHandler)
	handler := cors.Default().Handler(mux)

	err := http.ListenAndServe("localhost:1235", handler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func twitterAuthorizeURLHandler(w http.ResponseWriter, req *http.Request) {
	loginURL, err := proxy.HandleTwitterLoginURL()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
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
func twitterVerifyHandler(w http.ResponseWriter, req *http.Request) {
	userAddress := "0xE11BA2b4D45Eaed5996Cd0823791E0C93114882d"
	err := proxy.HandleTwitterVerify(userAddress)
	if err != nil {
		fmt.Println("proxy.HandleTwitterVerify error:", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Fprint(w, "verified")
}
