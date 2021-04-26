package auth

import (
	"fmt"

	"github.com/cli/oauth"
	"github.com/cli/oauth/api"
)

func displayCode(userCode string, uri string) error {
	fmt.Printf("Copy one-time code: %s\n", userCode)
	return nil
}

func browseURL(uri string) error {
	fmt.Println("Open %v in a browser and enter the code above when prompted.\n", uri)
	return nil
}

func GitHubHeadlessDeviceLogin(clientID string, scopes []string) (*api.AccessToken, error) {
	flow := &oauth.Flow{
		Hostname: "github.com",
		ClientID: clientID,
		Scopes:   scopes,
	}

	flow.DisplayCode = displayCode
	flow.BrowseURL = browseURL

	accessToken, err := flow.DeviceFlow()
	if err != nil {
		return nil, err
	}

	return accessToken, nil
}

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
