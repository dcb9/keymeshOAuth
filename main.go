package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dcb9/testOAuth/twitter"
)

var oauth1Config = twitter.NewConfig()

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch request.Path {
	case "/twitter/login-url":
		return getTwitterLoginURL()
	case "/twitter/user-info":
		return getTwitterUserInfo(request)
	case "/github/login-url":
		return getTwitterLoginURL()
	case "/github/user-info":
		return getTwitterUserInfo(request)
	}

	index, err := ioutil.ReadFile("public/index.html")
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(index),
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
	}, nil

}

func main() {
	lambda.Start(handler)
}

func getTwitterLoginURL() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Body:       twitter.GenerateTwitterLoginURL(oauth1Config),
		StatusCode: 200,
	}, nil
}

func getTwitterUserInfo(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	params := url.Values{}
	for k, v := range request.QueryStringParameters {
		params.Add(k, v)
	}
	req, _ := http.NewRequest(http.MethodGet, "?"+params.Encode(), nil)

	user := twitter.GetTwitterUser(oauth1Config, req)
	bytes, _ := json.Marshal(user)
	return events.APIGatewayProxyResponse{
		Body:       string(bytes),
		StatusCode: 200,
	}, nil
}
