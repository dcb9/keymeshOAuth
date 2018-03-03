package proxy

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"

	"github.com/dcb9/keymeshOAuth/db"
	"github.com/dcb9/keymeshOAuth/twitter"
)

var oauth1Config = twitter.NewConfig()

func HandleTwitterLoginURL() string {
	return twitter.GenerateTwitterLoginURL(oauth1Config)
}

var GetUserInfoErr = errors.New("get user info error")

func HandleTwitterCallback(req *http.Request) ([]byte, error) {
	user := twitter.GetTwitterUser(oauth1Config, req)
	if user == nil {
		return nil, GetUserInfoErr
	}

	userBytes, _ := json.Marshal(user)
	_, err := db.PutTwitterOAuthItem(*user)
	if err != nil {
		return nil, err
	}

	return userBytes, nil
}

type IndexHTMLData struct {
	TwitterAuthorizeURLApi template.URL
	TwitterCallbackURL     template.URL
}

func RenderIndexHTML(data IndexHTMLData, writer io.Writer) {
	t := template.Must(template.ParseFiles("public/index.html"))
	t.Execute(writer, data)
}
