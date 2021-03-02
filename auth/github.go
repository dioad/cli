package auth

import (
	"github.com/cli/oauth"
	"github.com/cli/oauth/api"
)

func GitHubDeviceLogin(clientID string, scopes []string) (*api.AccessToken, error) {
	flow := &oauth.Flow{
		Hostname: "github.com",
		ClientID: clientID,
		Scopes:   scopes,
	}

	accessToken, err := flow.DeviceFlow()
	if err != nil {
		return nil, err
	}

	return accessToken, nil
}
