package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dcb9/keymeshOAuth/eth"
	"github.com/dcb9/keymeshOAuth/proxy"
)

func main() {
	lambda.Start(corsHandler(errorHandler(handler)))
}

var (
	errPathNotMatch = errors.New("could not match any path")
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	switch request.Path {
	case "/oauth/twitter/authorize_url":
		return getTwitterAuthorizeURL()
	case "/oauth/twitter/callback":
		return twitterCallback(&request)
	case "/oauth/twitter/verify":
		return twitterVerify(&request)
	case "/users/search":
		return serializeUserInfoList(searchUsers(&request))
	case "/users":
		return serializeUserInfoList(getUsers(&request))
	case "/prekeys":
		return putPrekeys(&request)
	case "/account-info":
		return putAccountInfo(&request)
	case "/subscribe":
		return putAccountInfo(&request)
	}

	return events.APIGatewayProxyResponse{}, errPathNotMatch
}

var (
	errNoNetworkID      = errors.New(`"networkID" must be set`)
	errInvalidNetworkID = errors.New(`"networkID" must be a number`)
)

func requireNetworkID(request *events.APIGatewayProxyRequest) (int, error) {
	networkIDStr := request.QueryStringParameters["networkID"]
	if networkIDStr == "" {
		return 0, errNoNetworkID
	}
	networkID, err := strconv.Atoi(networkIDStr)
	if err != nil {
		return 0, errInvalidNetworkID
	}

	return networkID, nil
}

func putAccountInfo(request *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if request.HTTPMethod != http.MethodPut {
		return events.APIGatewayProxyResponse{}, fmt.Errorf(`Method "%s" is not allowed`, request.HTTPMethod)
	}

	if err := proxy.HandlePutAccountInfo(request.Body); err != nil {
		fmt.Println("proxy.HandlePutAccountInfo", err)
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
	}, nil
}

func putPrekeys(request *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	networkID, err := requireNetworkID(request)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	publicKeyHex := request.QueryStringParameters["publicKey"]
	if err = proxy.HandlePutPrekeys(publicKeyHex, networkID, request.Body); err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
	}, nil
}

var (
	errEmptyGetUsersParam = errors.New(`the query param "username" or "userAddress" must be set`)
)

func getUsers(request *events.APIGatewayProxyRequest) ([]*proxy.UserInfo, error) {
	networkID, err := requireNetworkID(request)
	if err != nil {
		return nil, err
	}

	username := request.QueryStringParameters["username"]
	if username != "" {
		userInfoList, err := proxy.HandleGetUserByUsername(username, networkID)
		return userInfoList, err
	}

	userAddress := request.QueryStringParameters["userAddress"]
	if userAddress != "" {
		userInfoList, err := proxy.HandleGetUserByUserAddress(userAddress, networkID)
		return userInfoList, err
	}

	return nil, errEmptyGetUsersParam
}

var (
	errEmptySearchUsersParam = errors.New(`the query param "usernamePrefix" must be set`)
)

func searchUsers(request *events.APIGatewayProxyRequest) ([]*proxy.UserInfo, error) {
	networkID, err := requireNetworkID(request)
	if err != nil {
		return nil, err
	}

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
		userInfoList, err := proxy.HandleSearchUserByUsernamePrefix(usernamePrefix, networkID, limit)
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

func twitterCallback(request *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
	errEmptyNetworkID   = errors.New("networkID could not be empty")
)

func twitterVerify(request *events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	networkID, err := requireNetworkID(request)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	userAddress := request.QueryStringParameters["userAddress"]
	if userAddress == "" {
		return events.APIGatewayProxyResponse{}, errEmptyUserAddress
	}
	var socialProof *proxy.SocialProof
	if eth.IsPrivateNetwork(networkID) {
		username := request.QueryStringParameters["username"]
		proofURL := request.QueryStringParameters["proofURL"]
		socialProof = &proxy.SocialProof{
			Username: username,
			ProofURL: proofURL,
		}
	}

	if err = proxy.HandleTwitterVerify(userAddress, networkID, socialProof); err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		Body:       "verified",
		StatusCode: 200,
	}, nil
}

type lambdaHandler func(events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func errorHandler(h lambdaHandler) lambdaHandler {
	return func(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		resp, err := h(request)

		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: http.StatusInternalServerError,
				Body:       err.Error(),
			}, nil
		}

		return resp, nil
	}
}

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
		}

		if resp.Headers == nil {
			resp.Headers = map[string]string{}
		}

		resp.Headers["Access-Control-Allow-Headers"] = "Accept, Accept-Language, Content-Language, Content-Type"
		resp.Headers["Access-Control-Allow-Methods"] = "GET, HEAD, POST, OPTIONS, PUT, DELETE, PATCH, CONNECT"
		resp.Headers["Access-Control-Allow-Origin"] = "*"
		resp.Headers["Vary"] = "Origin, Access-Control-Request-Method, Access-Control-Request-Headers"

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
