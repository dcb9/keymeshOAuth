package proxy

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/dcb9/keymeshOAuth/db"
	"github.com/dcb9/keymeshOAuth/twitter"
	"golang.org/x/crypto/ed25519"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/s3"
	goTwitter "github.com/dghubble/go-twitter/twitter"
)

var oauth1Config = twitter.NewConfig()
var prekeysBucketName = os.Getenv("PREKEYS_BUCKET_NAME")
var svc *s3.S3

func init() {
	sess, _ := session.NewSession()
	svc = s3.New(sess)
}

type GetUserLastProofEventPlayload struct {
	UserAddress string `json:"userAddress"`
	Platform    string `json:"platform"`
}

type SocialProof struct {
	ProofURL string `json:"proofURL"`
	Username string `json:"username"`
}

type omit *struct{}
type TwitterOAuthInfo struct {
	*goTwitter.User
	ContributorsEnabled omit `json:"contributors_enabled,omitempty"`
	CreatedAt           omit `json:"created_at,omitempty"`
	Email               omit `json:"email,omitempty"`
	Entities            omit `json:"entities,omitempty"`
	ID                  omit `json:"id,omitempty"`
	IDStr               omit `json:"id_str,omitempty"`
	Protected           omit `json:"protected,omitempty"`
	Status              omit `json:"status,omitempty"`
}

type UserInfo struct {
	UserAddress      string            `json:"userAddress"`
	Username         string            `json:"username"`
	PlatformName     db.PlatformName   `json:"platformName"`
	TwitterOAuthInfo *TwitterOAuthInfo `json:"twitterOAuthInfo"`
	GravatarHash     string            `json:"gravatarHash"`
}

type PutPrekeysReq struct {
	Signature string `json:"signature"`
	Prekeys   string `json:"prekeys"`
}

var (
	errInvalidSignature = errors.New("invalid signature")
)

func verifyPrekeys(publicKeyHex string, request *PutPrekeysReq) (err error) {
	publicKey, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return
	}

	signature := make([]byte, base64.StdEncoding.DecodedLen(len(request.Signature)))
	l, err := base64.StdEncoding.Decode(signature, []byte(request.Signature))
	if err != nil {
		return
	}
	signature = signature[:l]

	if !ed25519.Verify(publicKey, []byte(request.Prekeys), signature) {
		err = errInvalidSignature
		return
	}

	return
}

func HandlePutPrekeys(networkID, publicKeyHex string, requestBody string) (err error) {
	var req PutPrekeysReq
	err = json.Unmarshal([]byte(requestBody), &req)
	if err != nil {
		return
	}

	err = verifyPrekeys(publicKeyHex, &req)
	if err != nil {
		return
	}

	input := &s3.PutObjectInput{
		Body:   aws.ReadSeekCloser(strings.NewReader(requestBody)),
		Bucket: aws.String(prekeysBucketName),
		Key:    aws.String(fmt.Sprintf("%s/%s", networkID, publicKeyHex)),
	}
	_, err = svc.PutObject(input)
	return
}

func HandleSearchUserByUsernamePrefix(usernamePrefix string, limit int) ([]*UserInfo, error) {
	output, err := db.ScanUsernamePrefix(usernamePrefix)
	if err != nil {
		return nil, err
	}

	return convertScanUsernameOutput(output)
}

func NewTwitterOAuthInfo(user *goTwitter.User) *TwitterOAuthInfo {
	return &TwitterOAuthInfo{
		User: user,
	}
}

func fillTwitterOAuthInfo(userInfoList []*UserInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	usernames := make([]string, 0)
	for _, v := range userInfoList {
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

func HandleGetUserByUserAddress(userAddress string) ([]*UserInfo, error) {
	output, err := db.GetAuthorizationItemByUserAddress(&userAddress)
	if err != nil {
		return nil, err
	}

	userInfoList := make([]*UserInfo, 0)
	err = dynamodbattribute.UnmarshalListOfMaps(output.Items, &userInfoList)
	if err != nil {
		return nil, err
	}

	err = fillOAuthInfo(userInfoList)
	if err != nil {
		return nil, err
	}

	return userInfoList, nil
}

func HandleGetUserByUsername(username string) ([]*UserInfo, error) {
	output, err := db.ScanUsername(username)
	if err != nil {
		return nil, err
	}

	return convertScanUsernameOutput(output)
}

func convertScanUsernameOutput(output *dynamodb.ScanOutput) ([]*UserInfo, error) {
	userInfoList := make([]*UserInfo, 0)
	err := dynamodbattribute.UnmarshalListOfMaps(output.Items, &userInfoList)
	if err != nil {
		return nil, err
	}

	err = fillOAuthInfo(userInfoList)
	if err != nil {
		return nil, err
	}

	return userInfoList, nil
}

func HandleTwitterVerify(userAddress string) error {
	payload := GetUserLastProofEventPlayload{
		UserAddress: userAddress,
		Platform:    "twitter",
	}
	payloadBytes, _ := json.Marshal(payload)

	svc := lambda.New(session.New())
	input := &lambda.InvokeInput{
		FunctionName:   aws.String("getUserLastProofEventLambda"),
		Payload:        payloadBytes,
		InvocationType: aws.String("RequestResponse"),
	}

	result, err := svc.Invoke(input)
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

	fmt.Println(result)
	fmt.Printf("payload: %s\n", string(result.Payload))
	var socialProof SocialProof
	err = json.Unmarshal(result.Payload, &socialProof)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	item, err := db.GetTwitterOAuthItem(socialProof.Username)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	fmt.Println("getTwitterOAuthItem:", item)

	_, err = db.PutAuthorizationItem(db.AuthorizationItem{
		UserAddress:  userAddress,
		PlatformName: db.TwitterPlatformName,
		Username:     socialProof.Username,
		ProofURL:     socialProof.ProofURL,
		Verified:     true,
		VerifiedAt:   time.Now(),
	})
	if err != nil {
		return err
	}

	return nil
}

func HandleTwitterLoginURL() (string, error) {
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
