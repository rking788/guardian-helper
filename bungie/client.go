package bungie

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
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

// GetCurrentAccount will request the user info for the current user
// based on the OAuth token provided as part of the request.
func (c *Client) GetCurrentAccount() (*GetAccountResponse, error) {

	req, _ := http.NewRequest("GET", GetCurrentAccountEndpoint, nil)
	req.Header.Add("Content-Type", "application/json")
	c.AddAuthHeaders(req)

	itemsResponse, err := c.Do(req)
	if err != nil {
		fmt.Println("Failed to read the Items response from Bungie!: ", err.Error())
		return nil, err
	}
	defer itemsResponse.Body.Close()

	accountResponse := GetAccountResponse{}
	json.NewDecoder(itemsResponse.Body).Decode(&accountResponse)

	return &accountResponse, nil
}

// GetUserItems will make a request to the bungie API and retrieve all of the
// items for a specific Destiny membership ID. This includes all of their characters
// as well as the vault. The vault with have a character index of -1.
func (c *Client) GetUserItems(membershipType uint, membershipID string) (*ItemsEndpointResponse, error) {
	endpoint := fmt.Sprintf(ItemsEndpointFormat, membershipType, membershipID)

	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Add("Content-Type", "application/json")
	c.AddAuthHeaders(req)

	itemsResponse, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer itemsResponse.Body.Close()

	itemsJSON := &ItemsEndpointResponse{}
	json.NewDecoder(itemsResponse.Body).Decode(&itemsJSON)

	return itemsJSON, nil
}

// PostTransferItem is responsible for calling the Bungie.net API to transfer
// an item from a source to a destination. This could be either a user's character
// or the vault.
func (c *Client) PostTransferItem(body map[string]interface{}) {

	// TODO: This retry logic should probably be added to a middleware type function
	retry := true
	attempts := 0
	for {
		retry = false
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", TransferItemEndpointURL, strings.NewReader(string(jsonBody)))
		req.Header.Add("Content-Type", "application/json")
		c.AddAuthHeaders(req)

		resp, err := c.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()

		var response BaseResponse
		json.NewDecoder(resp.Body).Decode(&response)
		if response.ErrorCode == 36 || response.ErrorStatus == "ThrottleLimitExceededMomentarily" {
			time.Sleep(1 * time.Second)
			retry = true
		}

		fmt.Printf("Response for transfer request: %+v\n", response)
		attempts++
		if retry == false || attempts >= 5 {
			break
		}
	}
}
