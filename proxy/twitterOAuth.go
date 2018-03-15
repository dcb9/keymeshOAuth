package proxy

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/dcb9/keymeshOAuth/db"
	"github.com/dcb9/keymeshOAuth/twitter"
	goTwitter "github.com/dghubble/go-twitter/twitter"
)

var (
	oauth1Config   = twitter.NewConfig()
	GetUserInfoErr = errors.New("get user info error")
	lambdaService  = lambda.New(session.New())
)

type GetUserLastProofEventPlayload struct {
	UserAddress string          `json:"userAddress"`
	Platform    db.PlatformName `json:"platform"`
}

type SocialProof struct {
	ProofURL string `json:"proofURL"`
	Username string `json:"username"`
}

func HandleTwitterLoginURL() (string, error) {
	return twitter.GenerateTwitterLoginURL(oauth1Config)
}

func HandleTwitterCallback(req *http.Request) ([]byte, error) {
	user := twitter.GetTwitterUser(oauth1Config, req)
	if user == nil {
		return nil, GetUserInfoErr
	}

	_, err := db.PutTwitterOAuthItem(*user)
	if err != nil {
		return nil, err
	}

	return json.Marshal(user)
}

func HandleTwitterVerify(userAddress string, networkID int, socialProof *SocialProof) (err error) {
	if socialProof == nil {
		socialProof, err = getSocialProof(userAddress)
		if err != nil {
			return
		}
	}

	item, err := db.GetTwitterOAuthItem(socialProof.Username)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("getTwitterOAuthItem:", item)

	_, err = db.GetAuthorizationTable(networkID).
		PutAuthorizationItem(db.AuthorizationItem{
			UserAddress:  userAddress,
			PlatformName: db.TwitterPlatformName,
			Username:     socialProof.Username,
			ProofURL:     socialProof.ProofURL,
			Verified:     true,
			VerifiedAt:   time.Now(),
		})

	return
}

func getSocialProof(userAddress string) (*SocialProof, error) {
	payload := GetUserLastProofEventPlayload{
		UserAddress: userAddress,
		Platform:    db.TwitterPlatformName,
	}
	payloadBytes, _ := json.Marshal(payload)

	input := &lambda.InvokeInput{
		FunctionName:   aws.String("getUserLastProofEventLambda"),
		Payload:        payloadBytes,
		InvocationType: aws.String("RequestResponse"),
	}

	result, err := invokeLambda(input)

	var socialProof *SocialProof
	fmt.Printf("result payload: %s\n", string(result.Payload))
	err = json.Unmarshal(result.Payload, &socialProof)
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}

	return socialProof, nil
}

func NewTwitterOAuthInfo(user *goTwitter.User) *TwitterOAuthInfo {
	return &TwitterOAuthInfo{
		User: user,
	}
}

func fillTwitterOAuthInfo(userInfoList []*UserInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	usernames := make([]string, 0)
UserInfoList:
	for _, v := range userInfoList {
		for _, username := range usernames {
			if username == v.Username {
				continue UserInfoList
			}
		}
		usernames = append(usernames, v.Username)
	}
	if len(usernames) < 1 {
		return
	}

	data, err := db.BatchGetTwitterOAuth(usernames)
	if err != nil {
		panic(err)
	}

	list := make(map[string]*TwitterOAuthInfo)
	for i, v := range data {
		list[i] = NewTwitterOAuthInfo(&v)
	}
	for i, v := range userInfoList {
		if v.PlatformName == db.TwitterPlatformName {
			info := list[v.Username]
			userInfoList[i].TwitterOAuthInfo = info
			userInfoList[i].GravatarHash = fmt.Sprintf("%x", md5.Sum([]byte(info.User.Email)))
		}
	}
}

func fillOAuthInfo(userInfoList []*UserInfo) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("error %s", r))
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go fillTwitterOAuthInfo(userInfoList, &wg)
	//go fillFacebookOAuthInfo(userInfoList, &wg)
	//go fillGithubOAuthInfo(userInfoList, &wg)
	wg.Wait()

	return
}

func invokeLambda(input *lambda.InvokeInput) (result *lambda.InvokeOutput, err error) {
	result, err = lambdaService.Invoke(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case lambda.ErrCodeServiceException:
				fmt.Println(lambda.ErrCodeServiceException, aerr.Error())
			case lambda.ErrCodeResourceNotFoundException:
				fmt.Println(lambda.ErrCodeResourceNotFoundException, aerr.Error())
			case lambda.ErrCodeInvalidRequestContentException:
				fmt.Println(lambda.ErrCodeInvalidRequestContentException, aerr.Error())
			case lambda.ErrCodeInvalidRuntimeException:
				fmt.Println(lambda.ErrCodeInvalidRuntimeException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
	}

	return
}
