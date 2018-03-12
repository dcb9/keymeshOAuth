package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/dcb9/keymeshOAuth/proxy"
	"github.com/rs/cors"
)

var proxies struct {
	sync.RWMutex
	proxies map[int]*proxy.Proxy
}

func init() {
	proxies.proxies = make(map[int]*proxy.Proxy)
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/oauth/twitter/authorize_url", twitterAuthorizeURLHandler)
	mux.HandleFunc("/oauth/twitter/callback", twitterCallbackHandler)
	mux.HandleFunc("/oauth/twitter/verify", twitterVerifyHandler)
	mux.HandleFunc("/users/search", searchUsersHandler)
	mux.HandleFunc("/users", getUsersHandler)
	mux.HandleFunc("/prekeys", PutPrekeysHandler)

	entry := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		proxies.RLock()
		p, ok := proxies.proxies[networkID]
		proxies.RUnlock()
		if !ok {
			p = proxy.NewProxy(networkID)

			proxies.Lock()
			proxies.proxies[networkID] = p
			proxies.Unlock()
		}
		ctx := context.WithValue(r.Context(), "proxy", p)
		r = r.WithContext(ctx)

		mux.ServeHTTP(w, r)
	})

	handler := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "PUT", "POST", "DELETE", "OPTIONS"},
	}).Handler(entry)

	err := http.ListenAndServe("localhost:1235", handler)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func PutPrekeysHandler(w http.ResponseWriter, req *http.Request) {
	p := req.Context().Value("proxy").(*proxy.Proxy)
	publicKeyHex := req.Form.Get("publicKey")
	bytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		fmt.Println("ioutil.ReadAll", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	err = p.HandlePutPrekeys(publicKeyHex, string(bytes))
	if err != nil {
		fmt.Println("proxy.HandlePutPrekeys", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func getUsersHandler(w http.ResponseWriter, req *http.Request) {
	p := req.Context().Value("proxy").(*proxy.Proxy)
	err := req.ParseForm()
	if err != nil {
		fmt.Println("ParseForm:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	username := req.Form.Get("username")
	if username != "" {
		userInfoList, err := p.HandleGetUserByUsername(username)
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
		userInfoList, err := p.HandleGetUserByUserAddress(userAddress)
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
	p := req.Context().Value("proxy").(*proxy.Proxy)
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
		userInfoList, err := p.HandleSearchUserByUsernamePrefix(usernamePrefix, limit)
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
	p := req.Context().Value("proxy").(*proxy.Proxy)
	userBytes, err := p.HandleTwitterCallback(req)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	fmt.Fprint(w, string(userBytes))
}

func twitterVerifyHandler(w http.ResponseWriter, req *http.Request) {
	p := req.Context().Value("proxy").(*proxy.Proxy)
	err := req.ParseForm()
	if err != nil {
		fmt.Println("ParseForm:", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var socialProof *proxy.SocialProof
	if p.IsPrivateNetwork() {
		socialProof = &proxy.SocialProof{
			Username: req.Form.Get("username"),
			ProofURL: req.Form.Get("proofURL"),
		}
	}

	err = p.HandleTwitterVerify(req.Form.Get("userAddress"), socialProof)
	if err != nil {
		fmt.Println("proxy.HandleTwitterVerify error:", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Fprint(w, "verified")
}
