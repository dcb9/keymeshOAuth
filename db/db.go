package db

import (
	"errors"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type AuthorizationItem struct {
	UserAddress   string `dynamodbav:"user_address,omitempty"`
	RawTwitter    string `dynamodbav:"raw_twitter,omitempty"`
	TwitterEmail  string `dynamodbav:"twitter_email,omitempty"`
	RawGitHub     string `dynamodbav:"raw_github,omitempty"`
	GitHubEmail   string `dynamodbav:"github_email,omitempty"`
	RawFacebook   string `dynamodbav:"raw_facebook,omitempty"`
	FacebookEmail string `dynamodbav:"facebook_email,omitempty"`
}

var Conn *dynamodb.DynamoDB
var tableName = aws.String("authorizations")

func init() {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	Conn = dynamodb.New(sess, aws.NewConfig())
}

func PutAuthorizationItem(item AuthorizationItem) (*dynamodb.PutItemOutput, error) {
	item1, _ := dynamodbattribute.MarshalMap(item)
	putInput := &dynamodb.PutItemInput{
		Item:      item1,
		TableName: tableName,
	}

	return Conn.PutItem(putInput)
}

var emptyUserAddressErr = errors.New("user address can't be empty")

func GetAuthorizationItem(userAddress string) (*AuthorizationItem, error) {
	if userAddress == "" {
		return nil, emptyUserAddressErr
	}

	itemKey, _ := dynamodbattribute.MarshalMap(AuthorizationItem{
		UserAddress: userAddress,
	})
	//	to read an item from a table
	getItemInput := &dynamodb.GetItemInput{
		Key:       itemKey,
		TableName: tableName,
	}

	result, err := Conn.GetItem(getItemInput)
	if err != nil {
		return nil, err
	}
	item := AuthorizationItem{}
	_ = dynamodbattribute.UnmarshalMap(result.Item, &item)
	return &item, nil
}

func DynamoErrHandler(err error) {
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case dynamodb.ErrCodeConditionalCheckFailedException:
				log.Fatal(dynamodb.ErrCodeConditionalCheckFailedException, aerr.Error())
			case dynamodb.ErrCodeProvisionedThroughputExceededException:
				log.Fatal(dynamodb.ErrCodeProvisionedThroughputExceededException, aerr.Error())
			case dynamodb.ErrCodeResourceNotFoundException:
				log.Fatal(dynamodb.ErrCodeResourceNotFoundException, aerr.Error())
			case dynamodb.ErrCodeItemCollectionSizeLimitExceededException:
				log.Fatal(dynamodb.ErrCodeItemCollectionSizeLimitExceededException, aerr.Error())
			case dynamodb.ErrCodeInternalServerError:
				log.Fatal(dynamodb.ErrCodeInternalServerError, aerr.Error())
			default:
				log.Fatal(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Fatal(err.Error())
		}
		return
	}
}
