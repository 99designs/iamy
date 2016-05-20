package iamy

import "github.com/aws/aws-sdk-go/aws/session"

var sess *session.Session

func awsSession() *session.Session {
	if sess == nil {
		sess = session.New()
	}
	return sess
}
