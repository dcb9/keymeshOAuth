package proxy

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
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

type GetEthAddress struct {
	Username     string `json:"username"`
	PlatformName string `json:"platformName"`
	EthAddress   string `json:"ethAddress"`
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

func HandleSearchEthAddressesByUsernamePrefix(usernamePrefix string) ([]GetEthAddress, error) {
	output, err := db.ScanUsernamePrefix(usernamePrefix)
	if err != nil {
		return nil, err
	}

	return convertScanUsernameOutput(output)
}

func HandleSearchEthAddressesByUsername(username string) ([]GetEthAddress, error) {
	output, err := db.ScanUsername(username)
	if err != nil {
		return nil, err
	}

	return convertScanUsernameOutput(output)
}

func convertScanUsernameOutput(output *dynamodb.ScanOutput) ([]GetEthAddress, error) {
	var ethAddresses []GetEthAddress
	err := dynamodbattribute.UnmarshalListOfMaps(output.Items, &ethAddresses)
	if err != nil {
		return nil, err
	}

	return ethAddresses, nil
}

func HandleTwitterVerify(ethAddress string) error {
	payload := GetUserLastProofEventPlayload{
		UserAddress: ethAddress,
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
		EthAddress:   ethAddress,
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
