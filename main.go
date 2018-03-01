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
	case "/oauth/twitter/authorize_url":
		return getTwitterAuthorizeURL()
	case "/oauth/twitter/callback":
		return twitterCallback(request)
	}

	var content bytes.Buffer
	proxy.RenderIndexHTML(proxy.IndexHTMLData{
		TwitterAuthorizeURLApi: template.URL("/Prod/oauth/twitter/authorize_url"),
		TwitterCallbackURL:     template.URL("/Prod/oauth/twitter/callback"),
	}, &content)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       content.String(),
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
	}, nil

}

type lambdaHandler func(events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func main() {
	lambda.Start(corsHandler(handler))
}

func corsHandler(h lambdaHandler) lambdaHandler {
	return func(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		var resp events.APIGatewayProxyResponse
		var err error
		if request.HTTPMethod == "OPTIONS" {
			resp, err = events.APIGatewayProxyResponse{}, nil
		} else {
			resp, err = h(request)
		}
		if resp.Headers == nil {
			resp.Headers = map[string]string{}
		}

		resp.Headers["Access-Control-Allow-Headers"] = "Content-Type,X-Amz-Date,Authorization,X-Api-Key,X-Amz-Security-Token"
		resp.Headers["Access-Control-Allow-Methods"] = "*"
		resp.Headers["ccess-Control-Allow-Origin"] = "*"

		return resp, err
	}
}

func getTwitterAuthorizeURL() (events.APIGatewayProxyResponse, error) {
	url, err := proxy.HandleTwitterLoginURL()
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		Body:       url,
		StatusCode: 200,
	}, nil
}

func twitterCallback(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	params := url.Values{}
	for k, v := range request.QueryStringParameters {
		params.Add(k, v)
	}
	req, _ := http.NewRequest(http.MethodGet, "?"+params.Encode(), nil)

	userBytes, err := proxy.HandleTwitterCallback(req)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		Body:       string(userBytes),
		StatusCode: 200,
	}, nil
}
