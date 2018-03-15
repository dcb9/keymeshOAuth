package proxy

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var svc *s3.S3

func init() {
	sess, _ := session.NewSession()
	svc = s3.New(sess)
}
