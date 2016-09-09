package iamy

import (
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
)

var sess *session.Session

func awsSession() *session.Session {
	if sess == nil {
		var err error
		sess, err = session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		})

		if err != nil {
			log.Fatal("awsSession: couldn't create an AWS session", err)
		}
	}

	return sess
}
