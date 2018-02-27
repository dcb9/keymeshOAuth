package main

import (
	"bytes"
	"html/template"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dcb9/testOAuth/proxy"
)

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

	var content bytes.Buffer
	proxy.RenderIndexHTML(proxy.IndexHTMLData{
		GetTwitterAuthorizationURLApi: template.URL("twitter/login-url"),
		GetTwitterUserInfoApi:         template.URL("user-info"),
	}, &content)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       content.String(),
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
		Body:       proxy.HandleTwitterLoginURL(),
		StatusCode: 200,
	}, nil
}

func getTwitterUserInfo(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	params := url.Values{}
	for k, v := range request.QueryStringParameters {
		params.Add(k, v)
	}
	req, _ := http.NewRequest(http.MethodGet, "?"+params.Encode(), nil)

	userBytes, err := proxy.HandleTwitterUserInfo(req)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		Body:       string(userBytes),
		StatusCode: 200,
	}, nil
}
