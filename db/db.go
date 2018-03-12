package db

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	goTwitter "github.com/dghubble/go-twitter/twitter"
)

type AuthorizationItem struct {
	UserAddress  string       `json:"userAddress"`
	PlatformName PlatformName `json:"platformName"`
	Username     string       `json:"username"`
	ProofURL     string       `json:"proofURL"`
	Verified     bool         `json:"verified"`
	VerifiedAt   time.Time    `json:"verified_at"`
}

var conn *dynamodb.DynamoDB
var (
	authorizationTableName = os.Getenv("AUTHORIZATION_TABLE_NAME")
	twitterOAuthTableName  = os.Getenv("TWITTER_OAUTH_TABLE_NAME")
)

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
}

type DB struct {
	networkID int
}

func NewDB(networkID int) *DB {
	db := &DB{
		networkID: networkID,
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go db.tryToCreateAuthorizationTable(&wg)
	go db.tryToCreateTwitterOAuthTable(&wg)
	wg.Wait()

	return db
}

func (db *DB) tryToCreateAuthorizationTable(wg *sync.WaitGroup) {
	defer wg.Done()
	input := &dynamodb.CreateTableInput{
		TableName: db.getAuthorizationTableName(),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("userAddress"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("platformName"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("userAddress"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("platformName"),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	}
	conn.CreateTable(input)
}
func (db *DB) tryToCreateTwitterOAuthTable(wg *sync.WaitGroup) {
	defer wg.Done()
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(twitterOAuthTableName),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("screen_name"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("screen_name"),
				KeyType:       aws.String("HASH"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	}
	conn.CreateTable(input)
}

func (db *DB) BatchGetTwitterOAuth(screenNames []string) (map[string]goTwitter.User, error) {
	tableName := twitterOAuthTableName
	keys := make([]map[string]*dynamodb.AttributeValue, len(screenNames))
	for i, screenName := range screenNames {
		keys[i], _ = dynamodbattribute.MarshalMap(map[string]string{
			"screen_name": screenName,
		})
	}
	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]*dynamodb.KeysAndAttributes{
			tableName: &dynamodb.KeysAndAttributes{
				Keys: keys,
			},
		},
	}
	output, err := conn.BatchGetItem(input)
	if err != nil {
		return nil, err
	}

	items, ok := output.Responses[tableName]
	if !ok {
		return map[string]goTwitter.User{}, nil
	}

	typedItems := make([]goTwitter.User, len(items))
	err = dynamodbattribute.UnmarshalListOfMaps(items, &typedItems)
	if err != nil {
		return nil, err
	}

	mappedItems := make(map[string]goTwitter.User)
	for _, v := range typedItems {
		mappedItems[v.ScreenName] = v
	}

	return mappedItems, nil
}

func (db *DB) ScanUsername(username string) (*dynamodb.ScanOutput, error) {
	return db.scanUsername(username, aws.String("username = :username"))
}

func (db *DB) ScanUsernamePrefix(usernamePrefix string) (*dynamodb.ScanOutput, error) {
	return db.scanUsername(usernamePrefix, aws.String("begins_with(username, :username)"))
}

func (db *DB) scanUsername(username string, filterExpression *string) (*dynamodb.ScanOutput, error) {
	input := &dynamodb.ScanInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":username": {
				S: aws.String(username),
			},
		},
		FilterExpression: filterExpression,
		TableName:        db.getAuthorizationTableName(),
	}
	return conn.Scan(input)
}

func (db *DB) getAuthorizationTableName() *string {
	return aws.String(fmt.Sprintf("%s_%d", authorizationTableName, db.networkID))
}

func (db *DB) PutAuthorizationItem(item AuthorizationItem) (*dynamodb.PutItemOutput, error) {
	return db.putItem(item, db.getAuthorizationTableName())
}

func (db *DB) PutTwitterOAuthItem(user goTwitter.User) (*dynamodb.PutItemOutput, error) {
	return db.putItem(user, aws.String(twitterOAuthTableName))
}

func (db *DB) putItem(item interface{}, tableName *string) (*dynamodb.PutItemOutput, error) {
	_item, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return nil, err
	}

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

func (db *DB) GetAuthorizationItemByUserAddress(userAddress *string) (*dynamodb.QueryOutput, error) {
	input := &dynamodb.QueryInput{
		TableName:              db.getAuthorizationTableName(),
		KeyConditionExpression: aws.String("userAddress = :userAddress"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":userAddress": &dynamodb.AttributeValue{
				S: userAddress,
			},
		},
	}
	return conn.Query(input)
}

func (db *DB) GetTwitterOAuthItem(screenName string) (*dynamodb.GetItemOutput, error) {
	item := map[string]string{
		"screen_name": screenName,
	}
	return db.getItem(item, aws.String(twitterOAuthTableName))
}

func (db *DB) getItem(item interface{}, tableName *string) (*dynamodb.GetItemOutput, error) {
	_item, err := dynamodbattribute.MarshalMap(item)
	if err != nil {
		return nil, err
	}

	input := &dynamodb.GetItemInput{
		Key:       _item,
		TableName: tableName,
	}

	return conn.GetItem(input)
}
