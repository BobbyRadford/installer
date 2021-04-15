package ibmcloud

import (
	"context"

	"github.com/IBM-Cloud/bluemix-go/session"
)

// GetSession returns a IBM Cloud session
func GetSession(ctx context.Context) (*session.Session, error) {
	return session.New()
}
