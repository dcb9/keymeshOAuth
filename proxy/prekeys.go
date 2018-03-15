package proxy

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"golang.org/x/crypto/ed25519"
)

var prekeysBucketName = os.Getenv("PREKEYS_BUCKET_NAME")

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

func HandlePutPrekeys(publicKeyHex string, networkID int, requestBody string) (err error) {
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
		Key:    aws.String(fmt.Sprintf("%d/%s", networkID, publicKeyHex)),
	}
	_, err = svc.PutObject(input)
	return
}
