package bungie

import (
	"net/http"
)

// Client is a type that contains all information needed to make requests to the
// Bungie API.
type Client struct {
	*http.Client
	AccessToken string
	APIToken    string
}

// NewClient is a convenience function for creating a new Bungie.net Client that
// can be used to make requests to the API. This client shares the same
// http.Client for network requests instead of opening new connnections everytime.
func NewClient(accessToken, apiToken string) *Client {
	return &Client{
		Client:      http.DefaultClient,
		AccessToken: accessToken,
		APIToken:    apiToken,
	}
}

// AddAuthHeaders will handle adding the authentication headers from the
// current client to the specified Request.
func (c *Client) AddAuthHeaders(req *http.Request) {
	for key, val := range c.AuthenticationHeaders() {
		req.Header.Add(key, val)
	}
}

// AuthenticationHeaders will generate a map with the required headers to make
// an authenticated HTTP call to the Bungie API.
func (c *Client) AuthenticationHeaders() map[string]string {
	return map[string]string{
		"X-Api-Key":     c.APIToken,
		"Authorization": "Bearer " + c.AccessToken,
	}
}
