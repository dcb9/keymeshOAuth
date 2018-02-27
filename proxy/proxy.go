package proxy

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"

	"github.com/dcb9/testOAuth/db"
	"github.com/dcb9/testOAuth/twitter"
)

var oauth1Config = twitter.NewConfig()

func HandleTwitterLoginURL() string {
	return twitter.GenerateTwitterLoginURL(oauth1Config)
}

var GetUserInfoErr = errors.New("get user info error")

func HandleTwitterUserInfo(req *http.Request) ([]byte, error) {
	err := req.ParseForm()
	if err != nil {
		return nil, err
	}
	userAddress := req.Form.Get("user_address")

	user := twitter.GetTwitterUser(oauth1Config, req)
	if user == nil {
		return nil, GetUserInfoErr
	}

	userBytes, _ := json.Marshal(user)
	item := db.AuthorizationItem{
		UserAddress:  userAddress,
		PlatformName: db.TwitterPlatformName,
		OAuthData:    string(userBytes),
		Email:        user.Email,
	}
	_, err = db.PutAuthorizationItem(item)
	if err != nil {
		return nil, err
	}

	return userBytes, nil
}

type IndexHTMLData struct {
	GetTwitterAuthorizationURLApi template.URL
	GetTwitterUserInfoApi         template.URL
}

func RenderIndexHTML(data IndexHTMLData, writer io.Writer) {
	t := template.Must(template.ParseFiles("public/index.html"))
	t.Execute(writer, data)
}
