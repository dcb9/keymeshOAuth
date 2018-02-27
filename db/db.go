package db

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type AuthorizationItem struct {
	UserAddress  string       `dynamodbav:"user_address"`
	PlatformName PlatformName `dynamodbav:"platform_name"`
	OAuthData    string       `dynamodbav:"oauth_data"`
	Email        string       `dynamodbav:"email"`
}

var conn *dynamodb.DynamoDB
var tableName = aws.String(os.Getenv("AUTHORIZATION_TABLE_NAME"))

type PlatformName string

var (
	TwitterPlatformName  PlatformName = "twitter"
	FacebookPlatformName PlatformName = "facebook"
	GitHubPlatformName   PlatformName = "github"
)

func init() {
	sess, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	conn = dynamodb.New(sess, aws.NewConfig())

	// create table if not exists
}

func PutAuthorizationItem(item AuthorizationItem) (*dynamodb.PutItemOutput, error) {
	_item, _ := dynamodbattribute.MarshalMap(item)
	input := &dynamodb.PutItemInput{
		Item:      _item,
		TableName: tableName,
	}

	return conn.PutItem(input)
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
