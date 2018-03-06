package db

import (
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	goTwitter "github.com/dghubble/go-twitter/twitter"
)

type AuthorizationItem struct {
	// id = BuildItemId(platformName, original_id)
	UserAddress string    `json:"user_address"`
	ID          itemID    `json:"id"`
	Verified    bool      `json:"verified"`
	VerifiedAt  time.Time `json:"verified_at"`
}

var conn *dynamodb.DynamoDB
var (
	authorizationTableName = os.Getenv("AUTHORIZATION_TABLE_NAME")
	twitterOAuthTableName  = os.Getenv("TWITTER_OAUTH_TABLE_NAME")
)

type platformName string

var (
	TwitterPlatformName  platformName = "twitter"
	FacebookPlatformName platformName = "facebook"
	GitHubPlatformName   platformName = "github"
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
	return putItem(item, authorizationTableName)
}

func PutTwitterOAuthItem(user goTwitter.User) (*dynamodb.PutItemOutput, error) {
	return putItem(user, twitterOAuthTableName)
}

func putItem(item interface{}, tableName string) (*dynamodb.PutItemOutput, error) {
	_item, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.PutItemInput{
		Item:      _item,
		TableName: aws.String(tableName),
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

type itemID string

func BuildItemID(platformName platformName, originalId string) itemID {
	return itemID(string(platformName) + ":" + originalId)
}

func GetTwitterOAuthItem(screenName string) (*dynamodb.GetItemOutput, error) {
	item := map[string]string{
		"screen_name": screenName,
	}
	return getItem(item, twitterOAuthTableName)
}

func getItem(item interface{}, tableName string) (*dynamodb.GetItemOutput, error) {
	_item, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.GetItemInput{
		Key:       _item,
		TableName: aws.String(tableName),
	}

	return conn.GetItem(input)
}
