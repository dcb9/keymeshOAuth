package db

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var accountTableName = os.Getenv("ACCOUNT_TABLE_NAME")

type AccountInfo struct {
	UserAddress string    `json:"userAddress"`
	Name        string    `json:"name,omitempty"`
	Email       string    `json:"email"`
	Msg         string    `json:"msg,omitempty"`
	Sig         string    `json:"sig,omitempty"`
	ValidSig    bool      `json:"validSig,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

func PutAccountInfo(info AccountInfo) (*dynamodb.PutItemOutput, error) {
	return putItem(info, aws.String(accountTableName))
}

func tryToCreateAccountTable() {
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(accountTableName),
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("email"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("userAddress"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("email"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("userAddress"),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(5),
			WriteCapacityUnits: aws.Int64(5),
		},
	}
	output, err := conn.CreateTable(input)
	fmt.Println(output)
	fmt.Println(err)
}
