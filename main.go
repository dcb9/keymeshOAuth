package main

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dcb9/keymeshOAuth/proxy"
)

func main() {
	lambda.Start(corsHandler(handler))
}

var (
	errPathNotMatch = errors.New("could not match any path")
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch request.Path {
	case "/oauth/twitter/authorize_url":
		return getTwitterAuthorizeURL()
	case "/oauth/twitter/callback":
		return twitterCallback(request)
	case "/oauth/twitter/verify":
		return twitterVerify(request)
	case "/getEthAddresses":
		return getEthAddresses(request)
	case "/prekeys":
		return putPrekeys(request)
	}

	return events.APIGatewayProxyResponse{}, errPathNotMatch
}
func putPrekeys(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	networkID := request.QueryStringParameters["networkID"]
	publicKeyHex := request.QueryStringParameters["publicKey"]
	err := proxy.HandlePutPrekeys(networkID, publicKeyHex, request.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
	}, nil
}

var (
	errGetEthAddresses = errors.New(`the query param "username" or "usernamePrefix" must be set`)
)

func getEthAddresses(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	responseFunc := func(ethAddresses []proxy.GetEthAddress, err error) (events.APIGatewayProxyResponse, error) {
		return events.APIGatewayProxyResponse{}, nil
	}

	username := request.QueryStringParameters["username"]
	if username != "" {
		ethAddresses, err := proxy.HandleSearchEthAddressesByUsername(username)
		return responseFunc(ethAddresses, err)
	}

	usernamePrefix := request.QueryStringParameters["usernamePrefix"]
	if usernamePrefix != "" {
		ethAddresses, err := proxy.HandleSearchEthAddressesByUsernamePrefix(username)
		return responseFunc(ethAddresses, err)
	}

	return responseFunc(nil, errGetEthAddresses)
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

var (
	errEmptyEthAddress = errors.New("ethAddress could not be empty")
)

func twitterVerify(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ethAddress := request.QueryStringParameters["ethAddress"]
	if ethAddress == "" {
		return events.APIGatewayProxyResponse{}, errEmptyEthAddress
	}
	err := proxy.HandleTwitterVerify(ethAddress)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	return events.APIGatewayProxyResponse{
		Body:       "verified",
		StatusCode: 200,
	}, nil
}

type lambdaHandler func(events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

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

		resp.Headers["Access-Control-Allow-Headers"] = "*"
		resp.Headers["Access-Control-Allow-Methods"] = "*"
		resp.Headers["Access-Control-Allow-Origin"] = "*"

		return resp, err
	}
}
