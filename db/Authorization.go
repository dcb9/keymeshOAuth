package db

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var (
	authorizationTables    = make(map[int]*AuthorizationTable)
	authorizationTableName = os.Getenv("AUTHORIZATION_TABLE_NAME")
	rwmutex                = &sync.RWMutex{}
)

type AuthorizationItem struct {
	UserAddress  string       `json:"userAddress"`
	PlatformName PlatformName `json:"platformName"`
	Username     string       `json:"username"`
	ProofURL     string       `json:"proofURL"`
	Verified     bool         `json:"verified"`
	VerifiedAt   time.Time    `json:"verified_at"`
}

type AuthorizationTable struct {
	networkID int
}

func GetAuthorizationTable(networkID int) *AuthorizationTable {
	rwmutex.RLock()
	table, ok := authorizationTables[networkID]
	rwmutex.RUnlock()
	if !ok {
		table = &AuthorizationTable{
			networkID: networkID,
		}
		table.init()

		rwmutex.Lock()
		authorizationTables[networkID] = table
		rwmutex.Unlock()
	}

	return table
}

func (at *AuthorizationTable) init() {
	at.tryToCreateAuthorizationTable()
}

func (at *AuthorizationTable) PutAuthorizationItem(item AuthorizationItem) (*dynamodb.PutItemOutput, error) {
	return putItem(item, at.getAuthorizationTableName())
}

func (at *AuthorizationTable) GetAuthorizationItemByUserAddress(userAddress *string) (*dynamodb.QueryOutput, error) {
	input := &dynamodb.QueryInput{
		TableName:              at.getAuthorizationTableName(),
		KeyConditionExpression: aws.String("userAddress = :userAddress"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":userAddress": &dynamodb.AttributeValue{
				S: userAddress,
			},
		},
	}
	return conn.Query(input)
}

func (at *AuthorizationTable) ScanUsername(username string) (*dynamodb.ScanOutput, error) {
	return at.scanUsername(username, aws.String("username = :username"))
}

func (at *AuthorizationTable) ScanUsernamePrefix(usernamePrefix string) (*dynamodb.ScanOutput, error) {
	return at.scanUsername(usernamePrefix, aws.String("begins_with(username, :username)"))
}

func (at *AuthorizationTable) scanUsername(username string, filterExpression *string) (*dynamodb.ScanOutput, error) {
	input := &dynamodb.ScanInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":username": {
				S: aws.String(username),
			},
		},
		FilterExpression: filterExpression,
		TableName:        at.getAuthorizationTableName(),
	}
	return conn.Scan(input)
}

func (at *AuthorizationTable) getAuthorizationTableName() *string {
	return aws.String(fmt.Sprintf("%s_%d", authorizationTableName, at.networkID))
}

func (at *AuthorizationTable) tryToCreateAuthorizationTable() {
	input := &dynamodb.CreateTableInput{
		TableName: at.getAuthorizationTableName(),
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
