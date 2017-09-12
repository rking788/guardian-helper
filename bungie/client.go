package bungie

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BaseResponse represents the data returned as part of all of the Bungie API
// requests.
type BaseResponse struct {
	ErrorCode       int         `json:"ErrorCode"`
	ThrottleSeconds int         `json:"ThrottleSeconds"`
	ErrorStatus     string      `json:"ErrorStatus"`
	Message         string      `json:"Message"`
	MessageData     interface{} `json:"MessageData"`
}

// CurrentUserMembershipsResponse contains information about the membership data for the currently
// authorized user. The request for this information will use the access_token to determine the current user
// https://bungie-net.github.io/multi/operation_get_User-GetMembershipDataForCurrentUser.html#operation_get_User-GetMembershipDataForCurrentUser
type CurrentUserMembershipsResponse struct {
	*BaseResponse
	Response *struct {
		DestinyMemberships []*struct {
			DisplayName    string `json:"displayName"`
			MembershipType int    `json:"membershipType"`
			MembershipID   string `json:"membershipId"`
		} `json:"destinyMemberships"`
		BungieNetUser interface{} `json:"bungieNetUser"`
	} `json:"Response"`
}

// GetProfileResponse is the response from the GetProfile endpoint. This data contains information about
// the characeters, inventories, profile inventory, and equipped loadouts.
// https://bungie-net.github.io/multi/operation_get_Destiny2-GetProfile.html#operation_get_Destiny2-GetProfile
type GetProfileResponse struct {
	*BaseResponse
	Response *struct {
		ItemComponents *struct {
			Instances *struct {
				Data map[string]*ItemInstance `json:"data"`
			} `json:"instances"`
		} `json:"itemComponents"`
		Profile *struct {
			//https://bungie-net.github.io/multi/schema_Destiny-Entities-Profiles-DestinyProfileComponent.html#schema_Destiny-Entities-Profiles-DestinyProfileComponent
			Data *struct {
				UserInfo *struct {
					MembershipType int    `json:"membershipType"`
					MembershipID   string `json:"membershipId"`
					DisplayName    string `json:"displayName"`
				} `json:"userInfo"`
			} `json:"data"`
		} `json:"profile"`
		CharacterInventories *struct {
			Data map[string]*struct {
				Items ItemList `json:"items"`
			} `json:"data"`
		} `json:"characterInventories"`
		CharacterEquipment *struct {
			Data map[string]*struct {
				Items ItemList `json:"items"`
			} `json:"data"`
		} `json:"characterEquipment"`
		ProfileInventory *struct {
			Data *struct {
				Items ItemList `json:"Items"`
			} `json:"Data"`
		} `json:"profileInventory"`
		Characters *struct {
			Data CharacterMap `json:"data"`
		} `json:"Characters"`
		ProfileCurrencies *struct {
			Data *struct {
				Items ItemList `json:"items"`
			} `json:"data"`
		} `json:"profileCurrencies"`
	} `json:"Response"`
}

// D1GetAccountResponse is the response from a get current account API call
// this information needs to be used in all of the character/user specific endpoints.
type D1GetAccountResponse struct {
	Response *struct {
		DestinyMemberships []*struct {
			MembershipType int    `json:"membershipType"`
			DisplayName    string `json:"displayName"`
			MembershipID   string `json:"membershipId"`
		} `json:"destinyMemberships"`
	} `json:"Response"`
	Base *BaseResponse
}

// MembershipIDLookUpResponse represents the response to a Destiny membership ID lookup call
type MembershipIDLookUpResponse struct {
	Response        []*MembershipData `json:"Response"`
	ErrorCode       int               `json:"ErrorCode"`
	ThrottleSeconds int               `json:"ThrottleSeconds"`
	ErrorStatus     string            `json:"ErrorStatus"`
	Message         string            `json:"Message"`
	MessageData     interface{}       `json:"MessageData"`
}

// MembershipData represents the Response portion of the membership ID lookup
type MembershipData struct {
	MembershipID string `json:"membershipId"`
}

// ClientPool is a simple client buffer that will provided round robin access to a collection of Clients.
type ClientPool struct {
	Clients []*Client
	current int
}

// NewClientPool is a convenience initializer to create a new collection of Clients.
func NewClientPool() *ClientPool {

	addresses := readClientAddresses()
	clients := make([]*Client, 0, len(addresses))
	for _, addr := range addresses {
		client, err := NewCustomAddrClient(addr)
		if err != nil {
			fmt.Println("Error creating custom ipv6 client: ", err.Error())
			continue
		}

		clients = append(clients, client)
	}
	if len(clients) == 0 {
		clients = append(clients, &Client{Client: http.DefaultClient})
	}

	return &ClientPool{
		Clients: clients,
	}
}

// Get will return a pointer to the next Client that should be used.
func (pool *ClientPool) Get() *Client {
	c := pool.Clients[pool.current]
	if pool.current == (len(pool.Clients) - 1) {
		pool.current = 0
	} else {
		pool.current++
	}

	return c
}

func readClientAddresses() []string {
	// TODO: This should come from the environment or a file
	return []string{}
}

// Client is a type that contains all information needed to make requests to the
// Bungie API.
type Client struct {
	*http.Client
	Address     string
	AccessToken string
	APIToken    string
}

// NewCustomAddrClient will create a new Bungie Client instance with the provided local IP address.
func NewCustomAddrClient(address string) (*Client, error) {

	localAddr, err := net.ResolveIPAddr("ip6", address)
	if err != nil {
		return nil, err
	}

	localTCPAddr := net.TCPAddr{
		IP: localAddr.IP,
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			LocalAddr: &localTCPAddr,
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
	}

	httpClient := &http.Client{Transport: transport}

	return &Client{Client: httpClient, Address: address}, nil
}

// AddAuthValues will add the specified access token and api key to the provided client
func (c *Client) AddAuthValues(accessToken, apiKey string) {
	c.APIToken = apiKey
	c.AccessToken = accessToken
}

// AddAuthHeadersToRequest will handle adding the authentication headers from the
// current client to the specified Request.
func (c *Client) AddAuthHeadersToRequest(req *http.Request) {
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
func (c *Client) GetCurrentAccount() (*CurrentUserMembershipsResponse, error) {

	fmt.Println("Using client with local address: ", c.Address)

	req, _ := http.NewRequest("GET", GetMembershipsForCurrentUserEndpoint, nil)
	req.Header.Add("Content-Type", "application/json")
	c.AddAuthHeadersToRequest(req)

	membershipsResponse, err := c.Do(req)
	if err != nil {
		fmt.Println("Failed to read the Memberships response from Bungie!: ", err.Error())
		return nil, err
	}
	defer membershipsResponse.Body.Close()

	accountResponse := CurrentUserMembershipsResponse{}
	json.NewDecoder(membershipsResponse.Body).Decode(&accountResponse)

	return &accountResponse, nil
}

// GetUserProfileData is responsible for loading all of the profiles, characters, equipments, and inventories for all
// of the supplied user's characters.
func (c *Client) GetUserProfileData(membershipType int, membershipID string) (*GetProfileResponse, error) {

	fmt.Println("Using client with local address: ", c.Address)

	endpoint := fmt.Sprintf(GetProfileEndpointFormat, membershipType, membershipID)

	req, _ := http.NewRequest("GET", endpoint, nil)
	vals := url.Values{}
	vals.Add("components", strings.Join([]string{ProfilesComponent,
		ProfileInventoriesComponent, ProfileCurrenciesComponent, CharactersComponent,
		CharacterInventoriesComponent, CharacterEquipmentComponent, ItemInstancesComponent}, ","))

	// Add required headers and query string parameters
	req.Header.Add("Content-Type", "application/json")
	c.AddAuthHeadersToRequest(req)
	req.URL.RawQuery = vals.Encode()

	profileResponse, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer profileResponse.Body.Close()

	profile := &GetProfileResponse{}
	json.NewDecoder(profileResponse.Body).Decode(profile)

	return profile, nil
}

// GetUserItems will make a request to the bungie API and retrieve all of the
// items for a specific Destiny membership ID. This includes all of their characters
// as well as the vault. The vault with have a character index of -1.
func (c *Client) GetUserItems(membershipType int, membershipID string) (*D1ItemsEndpointResponse, error) {

	fmt.Println("Using client with local address: ", c.Address)

	endpoint := fmt.Sprintf(D1ItemsEndpointFormat, membershipType, membershipID)

	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Add("Content-Type", "application/json")
	c.AddAuthHeadersToRequest(req)

	itemsResponse, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer itemsResponse.Body.Close()

	itemsJSON := &D1ItemsEndpointResponse{}
	json.NewDecoder(itemsResponse.Body).Decode(&itemsJSON)

	return itemsJSON, nil
}

// PostTransferItem is responsible for calling the Bungie.net API to transfer
// an item from a source to a destination. This could be either a user's character
// or the vault.
func (c *Client) PostTransferItem(body map[string]interface{}) {

	fmt.Println("Using client with local address: ", c.Address)

	// TODO: This retry logic should probably be added to a middleware type function
	retry := true
	attempts := 0
	for {
		retry = false
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", TransferItemEndpointURL, strings.NewReader(string(jsonBody)))
		req.Header.Add("Content-Type", "application/json")
		c.AddAuthHeadersToRequest(req)

		resp, err := c.Do(req)
		if err != nil {
			fmt.Println("Error transferring item: ", err.Error())
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

// PostEquipItem is responsible for calling the Bungie.net API to equip
// an item on a specific character.
func (c *Client) PostEquipItem(body map[string]interface{}, isMultipleItems bool) {

	fmt.Println("Using client with local address: ", c.Address)
	// TODO: This retry logic should probably be added to a middleware type function
	retry := true
	attempts := 0
	for {
		retry = false
		jsonBody, _ := json.Marshal(body)

		endpoint := EquipSingleItemEndpointURL
		if isMultipleItems {
			endpoint = EquipMultiItemsEndpointURL
		}
		req, _ := http.NewRequest("POST", endpoint, strings.NewReader(string(jsonBody)))
		req.Header.Add("Content-Type", "application/json")
		c.AddAuthHeadersToRequest(req)

		resp, err := c.Do(req)
		if err != nil {
			fmt.Println("Error equipping item: ", err.Error())
			return
		}
		defer resp.Body.Close()

		var response BaseResponse
		json.NewDecoder(resp.Body).Decode(&response)
		if response.ErrorCode == 36 || response.ErrorStatus == "ThrottleLimitExceededMomentarily" {
			time.Sleep(1 * time.Second)
			retry = true
		}

		fmt.Printf("Response for equip request: %+v\n", response)
		attempts++
		if retry == false || attempts >= 5 {
			break
		}
	}
}
