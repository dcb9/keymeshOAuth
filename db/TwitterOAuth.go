package db

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	goTwitter "github.com/dghubble/go-twitter/twitter"
)

var twitterOAuthTableName = os.Getenv("TWITTER_OAUTH_TABLE_NAME")

type PlatformName string

var (
	TwitterPlatformName  PlatformName = "twitter"
	FacebookPlatformName PlatformName = "facebook"
	GitHubPlatformName   PlatformName = "github"
)

func GetTwitterOAuthItem(screenName string) (*dynamodb.GetItemOutput, error) {
	item := map[string]string{
		"screen_name": screenName,
	}
	return getItem(item, aws.String(twitterOAuthTableName))
}

func PutTwitterOAuthItem(user goTwitter.User) (*dynamodb.PutItemOutput, error) {
	return putItem(user, aws.String(twitterOAuthTableName))
}

func BatchGetTwitterOAuth(screenNames []string) (map[string]goTwitter.User, error) {
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

func tryToCreateTwitterOAuthTable() {
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
