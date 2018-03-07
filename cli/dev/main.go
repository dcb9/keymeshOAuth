package main

import (
	"encoding/json"
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
	mux.HandleFunc("/getEthAddresses", getEthAddressesHandler)
	handler := cors.Default().Handler(mux)

	err := http.ListenAndServe("localhost:1235", handler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func getEthAddressesHandler(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		fmt.Println("ParseForm:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	responseFunc := func(ethAddresses []proxy.GetEthAddress) {
		bs, _ := json.Marshal(ethAddresses)
		fmt.Fprint(w, string(bs))
	}

	username := req.Form.Get("username")
	if username != "" {
		ethAddresses, err := proxy.HandleSearchEthAddressesByUsername(username)
		if err != nil {
			fmt.Println("proxy.HandleSearchEthAddressesByUsername:", err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			responseFunc(ethAddresses)
		}
		return
	}

	usernamePrefix := req.Form.Get("usernamePrefix")
	if usernamePrefix != "" {
		ethAddresses, err := proxy.HandleSearchEthAddressesByUsernamePrefix(usernamePrefix)
		if err != nil {
			fmt.Println(err)
		} else {
			responseFunc(ethAddresses)
		}
		return
	}

	w.WriteHeader(http.StatusBadRequest)
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
	err := req.ParseForm()
	if err != nil {
		fmt.Println("ParseForm:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = proxy.HandleTwitterVerify(req.Form.Get("ethAddress"))
	if err != nil {
		fmt.Println("proxy.HandleTwitterVerify error:", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Fprint(w, "verified")
}
