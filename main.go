package main

import (
	"io/ioutil"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	switch request.Path {
	case "/twitter/login-url":
		return getTwitterLoginURL()
	case "/twitter/user-info":
		return getTwitterUserInfo()
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
		Body:       "http://fake.twitter.login.url",
		StatusCode: 200,
	}, nil
}

func getTwitterUserInfo() (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Body:       `{"foo": "bar"}`,
		StatusCode: 200,
	}, nil
}
