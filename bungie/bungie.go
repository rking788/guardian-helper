package bungie

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"bitbucket.org/rking788/guardian-helper/db"
	alexa "github.com/mikeflynn/go-alexa/skillserver"
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

// ItemsEndpointResponse represents the response from a call to the /Items endpoint
type ItemsEndpointResponse struct {
	Response *ItemsResponse `json:"Response"`
	Base     *BaseResponse
}

// ItemsResponse is the inner response from the /Items endpoint
type ItemsResponse struct {
	Data *ItemsData `json:"data"`
}

// ItemsData is the data attribute of the /Items response
type ItemsData struct {
	Items      []*Item      `json:"items"`
	Characters []*Character `json:"characters"`
}

// Item will represent a single inventory item returned by the /Items character
// endpoint.
type Item struct {
	ItemHash       uint   `json:"itemHash"`
	ItemID         string `json:"itemId"`
	Quantity       uint   `json:"quantity"`
	DamageType     uint   `json:"damageType"`
	DamageTypeHash uint   `json:"damageTypeHash"`
	//  IsGridComplete `json:"isGridComplete"`
	TransferStatus uint `json:"transferStatus"`
	State          uint `json:"state"`
	CharacterIndex int  `json:"characterIndex"`
	BucketHash     uint `json:"bucketHash"`
}

// Character will represent a single character entry returned by the /Items endpoint
type Character struct {
	CharacterBase *CharacterBase
	// NOTE: The rest is probably unused at least for the transferring items command
}

// CharacterBase represents the base data for a character entry
// returned by the /Items endpoint.
type CharacterBase struct {
	MembershipID           string    `json:"membershipId"`
	MembershipType         uint      `json:"membershipType"`
	CharacterID            string    `json:"characterId"`
	DateLastPlayed         time.Time `json:"dateLastPlayed"`
	PowerLevel             uint      `json:"powerLevel"`
	RaceHash               uint      `json:"raceHash"`
	GenderHash             uint      `json:"genderHash"`
	ClassHash              uint      `json:"classHash"`
	CurrentActivityHash    uint      `json:"currentActivityHash"`
	LastCompletedStoryHash uint      `json:"lastCompletedStoryHash"`
	GenderType             uint      `json:"genderType"`
	ClassType              uint      `json:"ClassType"`
}

// GetAccountResponse is the response from a get current account API call
// this information needs to be used in all of the character/user specific endpoints.
type GetAccountResponse struct {
	Response *struct {
		DestinyAccounts []*struct {
			UserInfo *struct {
				MembershipType uint   `json:"membershipType"`
				DisplayName    string `json:"displayName"`
				MembershipID   string `json:"membershipId"`
			} `json:"userInfo"`
		} `json:"destinyAccounts"`
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

// Client is a type that contains all information needed to make requests to the
// Bungie API.
type Client struct {
	*http.Client
	AccessToken string
	ApiToken    string
}

func NewClient(accessToken, apiToken string) *Client {
	return &Client{
		Client:      http.DefaultClient,
		AccessToken: accessToken,
		ApiToken:    apiToken,
	}
}

// AuthenticationHeaders will generate a map with the required headers to make
// an authenticated HTTP call to the Bungie API.
func (c *Client) AuthenticationHeaders() map[string]string {
	return map[string]string{
		"X-Api-Key":     c.ApiToken,
		"Authorization": "Bearer " + c.AccessToken,
	}
}

// MembershipIDFromDisplayName is responsible for retrieving the Destiny
// membership ID from the Bungie API given a specific display name
// from either Xbox or PSN
// TODO: This may no longer be needed as the GetCurrentAccount endpoint should fix all this.
func MembershipIDFromDisplayName(displayName string) string {

	endpoint := fmt.Sprintf(MembershipIDFromDisplayNameFormat, XBOX, displayName)
	client := NewClient("", os.Getenv("BUNGIE_API_KEY"))
	request, _ := http.NewRequest("GET", endpoint, nil)
	request.Header.Add("X-Api-Key", client.ApiToken)

	membershipResponse, err := client.Do(request)
	if err != nil {
		fmt.Println("Failed to request Destiny membership ID from Bungie!")
		return ""
	}

	membershipBytes, err := ioutil.ReadAll(membershipResponse.Body)
	if err != nil {
		fmt.Println("Couldn't read the response body from the Bungie API!")
		return ""
	}

	jsonResponse := MembershipIDLookUpResponse{}
	json.Unmarshal(membershipBytes, &jsonResponse)

	return jsonResponse.Response[0].MembershipID
}

// CountItem will count the number of the specified item and return an EchoResponse
// that can be serialized and sent back to the Alexa skill.
func CountItem(itemName, accessToken string) (*alexa.EchoResponse, error) {

	response := alexa.NewEchoResponse()

	client := NewClient(accessToken, os.Getenv("BUNGIE_API_KEY"))
	accountChan := make(chan *GetAccountResponse)

	go func() {
		currentAccount := GetCurrentAccount(client)
		accountChan <- currentAccount
	}()

	// Check common misinterpretations from Alexa
	if translation, ok := commonAlexaTranslations[itemName]; ok {
		itemName = translation
	}

	hash, err := db.GetItemHashFromName(itemName)
	if err != nil {
		outputStr := fmt.Sprintf("Sorry Guardian, I could not find any items named %s in your inventory.", itemName)
		response.OutputSpeech(outputStr)
		return response, nil
	}

	currentAccount := <-accountChan
	if currentAccount == nil {
		speech := fmt.Sprintf("Sorry Guardian, currently unable to get your account information.")
		response.OutputSpeech(speech)
		return response, nil
	}

	// TODO: Figure out how to support multiple accounts, meaning PSN and XBOX
	userInfo := currentAccount.Response.DestinyAccounts[0].UserInfo

	itemsJSON, err := GetUserItems(userInfo.MembershipType, userInfo.MembershipID, client)
	if err != nil {
		fmt.Println("Failed to read the Items response from Bungie!: ", err.Error())
		return nil, err
	}

	itemsData := itemsJSON.Response.Data
	matchingItems := itemsData.findItemsMatchingHash(hash)
	fmt.Printf("Found %d items entries in characters inventory.\n", len(matchingItems))

	if len(matchingItems) == 0 {
		outputStr := fmt.Sprintf("You don't have any %s on any of your characters.", itemName)
		response.OutputSpeech(outputStr)
		return response, nil
	}

	outputString := ""
	for _, item := range matchingItems {
		outputString += fmt.Sprintf("Your %s has %d %s. ", itemsData.characterClassNameAtIndex(item.CharacterIndex), item.Quantity, itemName)
	}
	response = response.OutputSpeech(outputString)

	return response, nil
}

func TransferItem(itemName, accessToken, sourceClass, destinationClass, countStr string) (*alexa.EchoResponse, error) {
	response := alexa.NewEchoResponse()

	client := NewClient(accessToken, os.Getenv("BUNGIE_API_KEY"))
	count := -1
	if countStr != "" {
		if tempCount, ok := strconv.Atoi(countStr); ok != nil {
			if tempCount <= 0 {
				output := fmt.Sprintf("Sorry Guardian, you need to specify a positive, non-zero count to be transferred, not %d", tempCount)
				fmt.Println(output)
				response.OutputSpeech(output)
				return response, nil
			}

			count = tempCount
		} else {
			response.OutputSpeech("Sorry Guardian, I didn't understand the number you asked to be transferred. If you don't specify a quantity then all will be transferred.")
			return response, nil
		}
	}

	// sourceHash := classNameToHash[sourceClass]
	// destinationHash := classNameToHash[destinationClass]
	// if sourceHash == 0 || destHash == 0 {
	// 	output := fmt.Sprintf("Sorry Guardian, I didn't understand the source (%s) or destination (%s) for the transfer.", sourceClass, destinationClass)
	// 	fmt.Println(output)
	// 	response.OutputSpeech(output)
	// 	return response, nil
	// }

	currentAccount := GetCurrentAccount(client)
	if currentAccount == nil {
		speech := fmt.Sprintf("Sorry Guardian, currently unable to get your account information.")
		response.OutputSpeech(speech)
		return response, nil
	}

	// Check common misinterpretations from Alexa
	if translation, ok := commonAlexaTranslations[itemName]; ok {
		itemName = translation
	}

	hash, err := db.GetItemHashFromName(itemName)
	if err != nil {
		outputStr := fmt.Sprintf("Sorry Guardian, I could not find any items named %s in your inventory.", itemName)
		response.OutputSpeech(outputStr)
		return response, nil
	}

	// TODO: Figure out how to support multiple accounts, meaning PSN and XBOx
	userInfo := currentAccount.Response.DestinyAccounts[0].UserInfo

	itemsJSON, err := GetUserItems(userInfo.MembershipType, userInfo.MembershipID, client)
	if err != nil {
		fmt.Println("Failed to read the Items response from Bungie!: ", err.Error())
		return nil, err
	}

	itemsData := itemsJSON.Response.Data
	matchingItems := itemsData.findItemsMatchingHash(hash)
	fmt.Printf("Found %d items entries in characters inventory.\n", len(matchingItems))

	if len(matchingItems) == 0 {
		outputStr := fmt.Sprintf("You don't have any %s on any of your characters.", itemName)
		response.OutputSpeech(outputStr)
		return response, nil
	}

	allChars := itemsJSON.Response.Data.Characters
	destCharacter, err := findDestinationCharacter(itemsJSON.Response.Data.Characters, destinationClass)
	if err != nil {
		output := fmt.Sprintf("Could not find a character with the specified class: %s", destinationClass)
		fmt.Println(output)
		response.OutputSpeech(output)
		return response, nil
	}

	transferItem(hash, matchingItems, allChars, destCharacter, userInfo.MembershipType, count, client)

	response.OutputSpeech("All set Guardian.")

	return response, nil
}

func transferItem(itemHash uint, itemSet []*Item, fullCharList []*Character, destCharacter *Character, membershipType uint, count int, client *Client) {

	var totalCount uint
	var wg sync.WaitGroup

	for _, item := range itemSet {

		if item.CharacterIndex == -1 && destCharacter == nil {
			continue
		} else if item.CharacterIndex != -1 && fullCharList[item.CharacterIndex] == destCharacter {
			fmt.Println("Attempting to transfer items to the same character... skipping")
			continue
		}

		totalCount += item.Quantity

		// If these items are already in the vault, skip it they will be transferred later
		if item.CharacterIndex == -1 {
			continue
		}

		wg.Add(1)

		go func(item *Item, charID string) {
			// These requests are all going TO the vault, the FROM the vault request
			// will go later for all of these.
			requestBody := map[string]interface{}{
				"itemReferenceHash": itemHash,
				"stackSize":         item.Quantity, // TODO: This should support transferring a subset
				"transferToVault":   true,
				"itemId":            item.ItemID,
				"characterId":       charID,
				"membershipType":    membershipType,
			}

			fmt.Printf("Transferring item: %+v\n", item)

			jsonBody, _ := json.Marshal(requestBody)
			fmt.Printf("Sending transfer request with body : %s\n", string(jsonBody))

			req, _ := http.NewRequest("POST", TransferItemEndpointURL, strings.NewReader(string(jsonBody)))
			req.Header.Add("Content-Type", "application/json")
			for key, val := range client.AuthenticationHeaders() {
				req.Header.Add(key, val)
			}

			resp, err := client.Do(req)
			if err != nil {
				return
			}

			respBytes, _ := ioutil.ReadAll(resp.Body)
			fmt.Printf("Response for transfer request: %s\n", string(respBytes))

			wg.Done()
		}(item, fullCharList[item.CharacterIndex].CharacterBase.CharacterID)
	}

	// Now transfer all of these items from the vault to the destination character
	if destCharacter == nil {
		// If the destination is the vault... then we are done already
		return
	}

	wg.Wait()

	requestBody := map[string]interface{}{
		"itemReferenceHash": itemHash,
		"stackSize":         totalCount, // TODO: This should support transferring a subset
		"transferToVault":   false,
		"itemId":            0,
		"characterId":       destCharacter.CharacterBase.CharacterID,
		"membershipType":    membershipType,
	}

	jsonBody, _ := json.Marshal(requestBody)
	fmt.Printf("Sending transfer request with body : %s\n", string(jsonBody))

	req, _ := http.NewRequest("POST", TransferItemEndpointURL, strings.NewReader(string(jsonBody)))
	req.Header.Add("Content-Type", "application/json")
	for key, val := range client.AuthenticationHeaders() {
		req.Header.Add(key, val)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending transfer request to Bungie API!")
		return
	}

	respBytes, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Response for transfer request: %s\n", string(respBytes))
}

func findDestinationCharacter(characters []*Character, class string) (*Character, error) {

	if class == "vault" {
		return nil, nil
	}

	destinationHash := classNameToHash[class]
	for _, char := range characters {
		if char.CharacterBase.ClassHash == destinationHash {
			return char, nil
		}
	}

	return nil, errors.New("could not find the specified destination character")
}

// GetCurrentAccount will request the user info for the current user
// based on the OAuth token provided as part of the request.
func GetCurrentAccount(client *Client) *GetAccountResponse {

	req, err := http.NewRequest("GET", GetCurrentAccountEndpoint, nil)
	req.Header.Add("Content-Type", "application/json")
	for key, val := range client.AuthenticationHeaders() {
		req.Header.Add(key, val)
	}

	itemsResponse, err := client.Do(req)
	itemsBytes, err := ioutil.ReadAll(itemsResponse.Body)
	if err != nil {
		fmt.Println("Failed to read the Items response from Bungie!: ", err.Error())
		return nil
	}

	accountResponse := GetAccountResponse{}
	json.Unmarshal(itemsBytes, &accountResponse)

	return &accountResponse
}

// GetUserItems will make a request to the bungie API and retrieve all of the
// items for a specific Destiny membership ID. This includes all of their characters
// as well as the vault. The vault with have a character index of -1.
func GetUserItems(membershipType uint, membershipID string, client *Client) (*ItemsEndpointResponse, error) {
	endpoint := fmt.Sprintf(ItemsEndpointFormat, membershipType, membershipID)

	req, _ := http.NewRequest("GET", endpoint, nil)
	req.Header.Add("Content-Type", "application/json")
	for key, val := range client.AuthenticationHeaders() {
		req.Header.Add(key, val)
	}

	itemsResponse, _ := client.Client.Do(req)
	itemsBytes, err := ioutil.ReadAll(itemsResponse.Body)
	if err != nil {
		return nil, err
	}

	itemsJSON := &ItemsEndpointResponse{}
	json.Unmarshal(itemsBytes, itemsJSON)

	return itemsJSON, nil
}

func (data *ItemsData) findItemsMatchingHash(itemHash uint) []*Item {
	result := make([]*Item, 0)

	for _, item := range data.Items {
		if item.ItemHash == itemHash {
			result = append(result, item)
		}
	}

	return result
}

func (data *ItemsData) characterClassNameAtIndex(index int) string {
	if index == -1 {
		return "Vault"
	} else if index >= len(data.Characters) {
		return "Unknown character"
	} else {
		return classHashToName[data.Characters[index].CharacterBase.ClassHash]
	}
}
