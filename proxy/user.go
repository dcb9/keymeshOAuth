package proxy

import (
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/dcb9/keymeshOAuth/db"
	goTwitter "github.com/dghubble/go-twitter/twitter"
)

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
	ProofURL         string            `json:"proofURL"`
}

func HandleSearchUserByUsernamePrefix(usernamePrefix string, networkID int, limit int) ([]*UserInfo, error) {
	output, err := db.GetAuthorizationTable(networkID).ScanUsernamePrefix(usernamePrefix)
	if err != nil {
		return nil, err
	}

	return convertScanUsernameOutput(output)
}

func HandleGetUserByUserAddress(userAddress string, networkID int) ([]*UserInfo, error) {
	output, err := db.GetAuthorizationTable(networkID).
		GetAuthorizationItemByUserAddress(&userAddress)
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

func HandleGetUserByUsername(username string, networkID int) ([]*UserInfo, error) {
	output, err := db.GetAuthorizationTable(networkID).
		ScanUsername(username)
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
