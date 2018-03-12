package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dcb9/keymeshOAuth/proxy"
)

var proxies struct {
	sync.RWMutex
	proxies map[int]*proxy.Proxy
}

func init() {
	proxies.proxies = make(map[int]*proxy.Proxy)
}

func main() {
	lambda.Start(corsHandler(errorHandler(handler)))
}

var (
	errPathNotMatch = errors.New("could not match any path")
	errNoNetworkID  = errors.New(`"networkID" must be set`)
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	networkIDStr := request.QueryStringParameters["networkID"]
	if networkIDStr == "" {
		return events.APIGatewayProxyResponse{}, errNoNetworkID
	}
	networkID, err := strconv.Atoi(networkIDStr)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
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

	switch request.Path {
	case "/oauth/twitter/authorize_url":
		return getTwitterAuthorizeURL()
	case "/oauth/twitter/callback":
		return twitterCallback(p, request)
	case "/oauth/twitter/verify":
		return twitterVerify(p, request)
	case "/users/search":
		return serializeUserInfoList(searchUsers(p, request))
	case "/users":
		return serializeUserInfoList(getUsers(p, request))
	case "/prekeys":
		return putPrekeys(p, request)
	}

	return events.APIGatewayProxyResponse{}, errPathNotMatch
}

func putPrekeys(p *proxy.Proxy, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	publicKeyHex := request.QueryStringParameters["publicKey"]
	err := p.HandlePutPrekeys(publicKeyHex, request.Body)
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

func getUsers(p *proxy.Proxy, request events.APIGatewayProxyRequest) ([]*proxy.UserInfo, error) {
	username := request.QueryStringParameters["username"]
	if username != "" {
		userInfoList, err := p.HandleGetUserByUsername(username)
		return userInfoList, err
	}

	userAddress := request.QueryStringParameters["userAddress"]
	if userAddress != "" {
		userInfoList, err := p.HandleGetUserByUserAddress(userAddress)
		return userInfoList, err
	}

	return nil, errEmptyGetUsersParam
}

var (
	errEmptySearchUsersParam = errors.New(`the query param "usernamePrefix" must be set`)
)

func searchUsers(p *proxy.Proxy, request events.APIGatewayProxyRequest) ([]*proxy.UserInfo, error) {
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
		userInfoList, err := p.HandleSearchUserByUsernamePrefix(usernamePrefix, limit)
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

func twitterCallback(p *proxy.Proxy, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	params := url.Values{}
	for k, v := range request.QueryStringParameters {
		params.Add(k, v)
	}
	req, _ := http.NewRequest(http.MethodGet, "?"+params.Encode(), nil)

	userBytes, err := p.HandleTwitterCallback(req)
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

func twitterVerify(p *proxy.Proxy, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userAddress := request.QueryStringParameters["userAddress"]
	if userAddress == "" {
		return events.APIGatewayProxyResponse{}, errEmptyUserAddress
	}
	var socialProof *proxy.SocialProof
	if p.IsPrivateNetwork() {
		username := request.QueryStringParameters["username"]
		proofURL := request.QueryStringParameters["proofURL"]
		socialProof = &proxy.SocialProof{
			Username: username,
			ProofURL: proofURL,
		}
	}

	err := p.HandleTwitterVerify(userAddress, socialProof)
	if err != nil {
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
