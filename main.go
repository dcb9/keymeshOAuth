package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"

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
	case "/users/search":
		return serializeUserInfoList(searchUsers(request))
	case "/users":
		return serializeUserInfoList(getUsers(request))
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
	errEmptyGetUsersParam = errors.New(`the query param "username" or "userAddress" must be set`)
)

func getUsers(request events.APIGatewayProxyRequest) ([]*proxy.UserInfo, error) {
	username := request.QueryStringParameters["username"]
	if username != "" {
		userInfoList, err := proxy.HandleGetUserByUsername(username)
		return userInfoList, err
	}

	userAddress := request.QueryStringParameters["userAddress"]
	if userAddress != "" {
		userInfoList, err := proxy.HandleGetUserByUserAddress(userAddress)
		return userInfoList, err
	}

	return nil, errEmptyGetUsersParam
}

var (
	errEmptySearchUsersParam = errors.New(`the query param "usernamePrefix" must be set`)
)

func searchUsers(request events.APIGatewayProxyRequest) ([]*proxy.UserInfo, error) {
	limit := 10
	limitStr := request.QueryStringParameters["limit"]
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return nil, err
		}
	}

	usernamePrefix := request.QueryStringParameters["usernamePrefix"]
	if usernamePrefix != "" {
		userInfoList, err := proxy.HandleSearchUserByUsernamePrefix(usernamePrefix, limit)
		return userInfoList, err
	}

	return nil, errEmptySearchUsersParam
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
	errEmptyUserAddress = errors.New("userAddress could not be empty")
)

func twitterVerify(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userAddress := request.QueryStringParameters["userAddress"]
	if userAddress == "" {
		return events.APIGatewayProxyResponse{}, errEmptyUserAddress
	}
	err := proxy.HandleTwitterVerify(userAddress)
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
			resp, err = events.APIGatewayProxyResponse{
				StatusCode: 200,
			}, nil
		} else {
			resp, err = h(request)
			if err != nil && resp.StatusCode == 0 {
				resp.StatusCode = http.StatusInternalServerError
			}
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

func serializeUserInfoList(userInfoList []*proxy.UserInfo, err error) (events.APIGatewayProxyResponse, error) {
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
		}, err
	}
	bs, _ := json.Marshal(userInfoList)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(bs),
	}, nil
}
