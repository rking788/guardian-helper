package bungie

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/kpango/glg"
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
		BungieNetUser *struct {
			MembershipID string `json:"membershipId"`
		} `json:"bungieNetUser"`
	} `json:"Response"`
}

// GetProfileResponse is the response from the GetProfile endpoint. This data contains information about
// the characeters, inventories, profile inventory, and equipped loadouts.
// https://bungie-net.github.io/multi/operation_get_Destiny2-GetProfile.html#operation_get_Destiny2-GetProfile
type GetProfileResponse struct {
	*BaseResponse
	Response *struct {
		CharacterInventories *CharacterMappedItemListData `json:"characterInventories"`
		CharacterEquipment   *CharacterMappedItemListData `json:"characterEquipment"`
		ProfileInventory     *ItemListData                `json:"profileInventory"`
		ProfileCurrencies    *ItemListData                `json:"profileCurrencies"`
		ItemComponents       *struct {
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
		Characters *struct {
			Data CharacterMap `json:"data"`
		} `json:"Characters"`
	} `json:"Response"`
}

type ItemListData struct {
	Data *struct {
		Items ItemList `json:"items"`
	} `json:"data"`
}

type CharacterMappedItemListData struct {
	Data map[string]*struct {
		Items ItemList `json:"items"`
	} `json:"data"`
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
			glg.Errorf("Error creating custom ipv6 client: %s", err.Error())
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

func readClientAddresses() (result []string) {
	result = make([]string, 0, 32)

	in, err := os.OpenFile("local_clients.txt", os.O_RDONLY, 0644)
	if err != nil {
		glg.Warn("Local clients list does not exist, using the default...")
		return
	}

	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		addr := scanner.Text()
		if addr != "" {
			result = append(result, addr)
		}
	}

	if err != nil {
		glg.Errorf("Failed to read local clients: %s", err.Error())
	}

	return
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

	glg.Debugf("Client with local address: %s", c.Address)

	req, _ := http.NewRequest("GET", GetMembershipsForCurrentUserEndpoint, nil)
	req.Header.Add("Content-Type", "application/json")
	c.AddAuthHeadersToRequest(req)

	membershipsResponse, err := c.Do(req)
	if err != nil {
		glg.Errorf("Failed to read the Memberships response from Bungie!: %s", err.Error())
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

	glg.Debugf("Client local address: %s", c.Address)

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

func (c *Client) GetCurrentEquipment(membershipType int, membershipID string) (*GetProfileResponse, error) {

	glg.Debugf("Client local address: %s", c.Address)

	endpoint := fmt.Sprintf(GetProfileEndpointFormat, membershipType, membershipID)

	req, _ := http.NewRequest("GET", endpoint, nil)
	vals := url.Values{}
	vals.Add("components", strings.Join([]string{CharactersComponent, CharacterEquipmentComponent, ItemInstancesComponent}, ","))

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

// PostTransferItem is responsible for calling the Bungie.net API to transfer
// an item from a source to a destination. This could be either a user's character
// or the vault.
func (c *Client) PostTransferItem(body map[string]interface{}) {

	glg.Debugf("Client local address: %s", c.Address)

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
			glg.Errorf("Error transferring item: %s", err.Error())
			return
		}
		defer resp.Body.Close()

		var response BaseResponse
		json.NewDecoder(resp.Body).Decode(&response)
		if response.ErrorCode == 36 || response.ErrorStatus == "ThrottleLimitExceededMomentarily" {
			time.Sleep(1 * time.Second)
			retry = true
		}

		glg.Infof("Response for transfer request: %+v", response)
		attempts++
		if retry == false || attempts >= 5 {
			break
		}
	}
}

// PostEquipItem is responsible for calling the Bungie.net API to equip
// an item on a specific character.
func (c *Client) PostEquipItem(body map[string]interface{}, isMultipleItems bool) {

	glg.Debugf("Client local address: %s", c.Address)
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
			glg.Errorf("Error equipping item: %s", err.Error())
			return
		}
		defer resp.Body.Close()

		var response BaseResponse
		json.NewDecoder(resp.Body).Decode(&response)
		if response.ErrorCode == 36 || response.ErrorStatus == "ThrottleLimitExceededMomentarily" {
			time.Sleep(1 * time.Second)
			retry = true
		}

		glg.Infof("Response for equip request: %+v", response)
		attempts++
		if retry == false || attempts >= 5 {
			break
		}
	}
}
