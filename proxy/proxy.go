package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"

	"github.com/dcb9/keymeshOAuth/db"
	"github.com/dcb9/keymeshOAuth/twitter"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

var oauth1Config = twitter.NewConfig()

type GetUserLastProofEventPlayload struct {
	UserAddress string `json:"userAddress"`
	Platform    string `json:"platform"`
}

type SocialProof struct {
	ProofURL string `json:"proofURL"`
	Username string `json:"username"`
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
		UserAddress: userAddress,
		ID:          db.BuildItemID(db.TwitterPlatformName, socialProof.Username),
		Verified:    true,
		VerifiedAt:  time.Now(),
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

type IndexHTMLData struct {
	TwitterAuthorizeURLApi template.URL
	TwitterCallbackURL     template.URL
}

func RenderIndexHTML(data IndexHTMLData, writer io.Writer) {
	t := template.Must(template.ParseFiles("public/index.html"))
	t.Execute(writer, data)
}
