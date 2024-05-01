package auth

import (
	"fmt"

	"github.com/cli/oauth"
	"github.com/cli/oauth/api"
)

type DeviceLoginFunc func(string, []string) (*api.AccessToken, error)

func displayCode(userCode string, uri string) error {
	fmt.Printf("One-time code: %s\n", userCode)
	return nil
}

func browseURL(uri string) error {
	fmt.Printf("Go to %v in a browser and enter the code above when prompted.\n", uri)
	return nil
}

func gitHubHeadlessDeviceLogin(clientID string, scopes []string) (*api.AccessToken, error) {

	host := oauth.GitHubHost("github.com")

	flow := &oauth.Flow{
		Host:     host,
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

func gitHubDeviceLogin(clientID string, scopes []string) (*api.AccessToken, error) {
	host := oauth.GitHubHost("github.com")
	flow := &oauth.Flow{
		Host:     host,
		ClientID: clientID,
		Scopes:   scopes,
	}

	accessToken, err := flow.DeviceFlow()
	if err != nil {
		return nil, err
	}

	return accessToken, nil
}

func GitHubDeviceLogin(clientID string, scopes []string, headless bool) (*api.AccessToken, error) {
	var deviceLoginFunc DeviceLoginFunc
	if headless {
		deviceLoginFunc = gitHubHeadlessDeviceLogin
	} else {
		deviceLoginFunc = gitHubDeviceLogin
	}

	return deviceLoginFunc(clientID, scopes)
}
