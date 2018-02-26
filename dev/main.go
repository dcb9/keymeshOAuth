package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/dcb9/testOAuth/db"
	"github.com/dcb9/testOAuth/twitter"
)

var oauth1Config = twitter.NewConfig()

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
	loginURL := twitter.GenerateTwitterLoginURL(oauth1Config)
	fmt.Fprint(w, loginURL)
}
func twitterUserInfoHandler(w http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		return
	}
	userAddress := req.Form.Get("user_address")

	user := twitter.GetTwitterUser(oauth1Config, req)
	bytes, _ := json.Marshal(user)

	item, err := db.GetAuthorizationItem(userAddress)
	if err != nil {
		return
	}
	if item.UserAddress == "" {
		// new item
		item.UserAddress = userAddress
	}
	item.RawTwitter = string(bytes)

	item.TwitterEmail = user.Email

	_, err = db.PutAuthorizationItem(*item)
	if err != nil {
		return
	}
	fmt.Fprint(w, string(bytes))
}

// welcomeHandler shows a welcome message and login button.
func welcomeHandler(w http.ResponseWriter, req *http.Request) {
	type templateStruct struct {
		GetTwitterAuthorizationURLApi template.URL
		GetTwitterUserInfoApi         template.URL
		TwitterBindUserApi            template.URL
	}

	t := template.Must(template.ParseFiles("public/index.html"))
	t.Execute(w, templateStruct{
		GetTwitterAuthorizationURLApi: template.URL("/twitter/login-url"),
		GetTwitterUserInfoApi:         template.URL("/twitter/user-info"),
		TwitterBindUserApi:            template.URL("/twitter/bind-user"),
	})
	/*
		t.Execute(w, templateStruct{
			GetTwitterAuthorizationURLApi: template.URL("https://cors-anywhere.herokuapp.com/https://lhql95dprb.execute-api.ap-northeast-1.amazonaws.com/Prod/twitter/login-url"),
			GetTwitterUserInfoApi:         template.URL("https://cors-anywhere.herokuapp.com/https://lhql95dprb.execute-api.ap-northeast-1.amazonaws.com/Prod/twitter/user-info"),
		})
	*/
}
