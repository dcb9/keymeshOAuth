package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/dcb9/keymeshOAuth/eth"
	"github.com/dcb9/keymeshOAuth/proxy"
	"github.com/rs/cors"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/oauth/twitter/authorize_url", twitterAuthorizeURLHandler)
	mux.HandleFunc("/oauth/twitter/callback", twitterCallbackHandler)
	mux.HandleFunc("/oauth/twitter/verify", requireNetworkID(twitterVerifyHandler))

	mux.HandleFunc("/users/search", requireNetworkID(searchUsersHandler))
	mux.HandleFunc("/users", requireNetworkID(getUsersHandler))

	mux.HandleFunc("/prekeys", requireNetworkID(PutPrekeysHandler))
	mux.HandleFunc("/account-info", putAccountInfoHandler)
	mux.HandleFunc("/subscribe", putAccountInfoHandler)

	handler := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "PUT", "POST", "DELETE", "OPTIONS"},
	}).Handler(mux)

	err := http.ListenAndServe("localhost:1235", handler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func getNetworkID(req *http.Request) int {
	return req.Context().Value("networkID").(int)
}

func requireNetworkID(handler func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		networkIDStr := r.Form.Get("networkID")
		if networkIDStr == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "networkID must be set")
			return
		}
		networkID, err := strconv.Atoi(networkIDStr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, err.Error())
			return
		}

		ctx := context.WithValue(r.Context(), "networkID", networkID)
		r = r.WithContext(ctx)

		handler(w, r)
	}
}

func putAccountInfoHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPut {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("ioutil.ReadAll", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = proxy.HandlePutAccountInfo(string(bytes))
	if err != nil {
		fmt.Println("proxy.HandlePutAccountInfo", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func PutPrekeysHandler(w http.ResponseWriter, req *http.Request) {
	networkID := getNetworkID(req)
	publicKeyHex := req.Form.Get("publicKey")
	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("ioutil.ReadAll", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = proxy.HandlePutPrekeys(publicKeyHex, networkID, string(bytes))
	if err != nil {
		fmt.Println("proxy.HandlePutPrekeys", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func getUsersHandler(w http.ResponseWriter, req *http.Request) {
	networkID := getNetworkID(req)
	err := req.ParseForm()
	if err != nil {
		fmt.Println("ParseForm:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	username := req.Form.Get("username")
	if username != "" {
		userInfoList, err := proxy.HandleGetUserByUsername(username, networkID)
		if err != nil {
			fmt.Println("proxy.HandleGetUserByUsername:", err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			bs, _ := json.Marshal(userInfoList)
			fmt.Fprint(w, string(bs))
		}
		return
	}

	userAddress := req.Form.Get("userAddress")
	if userAddress != "" {
		userInfoList, err := proxy.HandleGetUserByUserAddress(userAddress, networkID)
		if err != nil {
			fmt.Println("proxy.HandleGetUserByUserAddress:", err)
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			bs, _ := json.Marshal(userInfoList)
			fmt.Fprint(w, string(bs))
		}
		return
	}

	w.WriteHeader(http.StatusBadRequest)
}

func searchUsersHandler(w http.ResponseWriter, req *http.Request) {
	networkID := getNetworkID(req)
	err := req.ParseForm()
	if err != nil {
		fmt.Println("ParseForm:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	usernamePrefix := req.Form.Get("usernamePrefix")
	limitStr := req.Form.Get("limit")
	limit := 10
	if limitStr != "" {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	if usernamePrefix != "" {
		userInfoList, err := proxy.HandleSearchUserByUsernamePrefix(usernamePrefix, networkID, limit)
		if err != nil {
			fmt.Println(err)
		} else {
			bs, _ := json.Marshal(userInfoList)
			fmt.Fprint(w, string(bs))
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
	networkID := getNetworkID(req)
	err := req.ParseForm()
	if err != nil {
		fmt.Println("ParseForm:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var socialProof *proxy.SocialProof
	if eth.IsPrivateNetwork(networkID) {
		socialProof = &proxy.SocialProof{
			Username: req.Form.Get("username"),
			ProofURL: req.Form.Get("proofURL"),
		}
	}

	err = proxy.HandleTwitterVerify(req.Form.Get("userAddress"), networkID, socialProof)
	if err != nil {
		fmt.Println("proxy.HandleTwitterVerify error:", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Fprint(w, "verified")
}
