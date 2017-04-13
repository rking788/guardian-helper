package bungie

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
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

// MembershipIDFromDisplayName is responsible for retrieving the Destiny
// membership ID from the Bungie API given a specific display name
// from either Xbox or PSN
// TODO: This may no longer be needed as the GetCurrentAccount endpoint should fix all this.
func MembershipIDFromDisplayName(displayName string) string {

	endpoint := fmt.Sprintf(MembershipIDFromDisplayNameFormat, XBOX, displayName)
	client := NewClient("", os.Getenv("BUNGIE_API_KEY"))
	request, _ := http.NewRequest("GET", endpoint, nil)
	request.Header.Add("X-Api-Key", client.APIToken)

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

	// Load all items on all characters
	itemsChannel := make(chan *AllItemsMsg)
	go GetAllItemsForCurrentUser(client, itemsChannel)

	startHash := time.Now()
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

	fmt.Println("Time to get translation and hash from DB: ", time.Since(startHash))

	itemsJSON, _ := <-itemsChannel
	if itemsJSON.error != nil {
		response.OutputSpeech("Sorry Guardian, there was an error reading your current account information.")
		return response, nil
	}
	startFindItemsMatching := time.Now()
	itemsData := itemsJSON.ItemsEndpointResponse.Response.Data
	matchingItems := itemsData.findItemsMatchingHash(hash)
	fmt.Printf("Found %d items entries in characters inventory.\n", len(matchingItems))

	if len(matchingItems) == 0 {
		outputStr := fmt.Sprintf("You don't have any %s on any of your characters.", itemName)
		response.OutputSpeech(outputStr)
		return response, nil
	}
	time.Since(startFindItemsMatching)

	formOutput := time.Now()
	outputString := ""
	for _, item := range matchingItems {
		outputString += fmt.Sprintf("Your %s has %d %s. ", itemsData.characterClassNameAtIndex(item.CharacterIndex), item.Quantity, itemName)
	}
	response = response.OutputSpeech(outputString)
	fmt.Println("Time to form output: ", time.Since(formOutput))

	return response, nil
}

// TransferItem is responsible for calling the necessary Bungie.net APIs to
// transfer the specified item to the specified character. The quantity is optional
// as well as the source class. If no quantity is specified, all of the specific
// items will be transfered to the particular character.
func TransferItem(itemName, accessToken, sourceClass, destinationClass string, count int) (*alexa.EchoResponse, error) {
	response := alexa.NewEchoResponse()

	client := NewClient(accessToken, os.Getenv("BUNGIE_API_KEY"))

	itemsChannel := make(chan *AllItemsMsg)
	go GetAllItemsForCurrentUser(client, itemsChannel)

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

	itemsJSON := <-itemsChannel
	if itemsJSON.error != nil {
		fmt.Println("Failed to read the Items response from Bungie!: ", err.Error())
		return nil, err
	}

	itemsData := itemsJSON.ItemsEndpointResponse.Response.Data
	matchingItems := itemsData.findItemsMatchingHash(hash)
	fmt.Printf("Found %d items entries in characters inventory.\n", len(matchingItems))

	if len(matchingItems) == 0 {
		outputStr := fmt.Sprintf("You don't have any %s on any of your characters.", itemName)
		response.OutputSpeech(outputStr)
		return response, nil
	}

	allChars := itemsJSON.ItemsEndpointResponse.Response.Data.Characters
	destCharacter, err := findDestinationCharacter(allChars, destinationClass)
	if err != nil {
		output := fmt.Sprintf("Could not find a character with the specified class: %s", destinationClass)
		fmt.Println(output)
		response.OutputSpeech(output)
		return response, nil
	}

	transferItem(hash, matchingItems, allChars, destCharacter,
		itemsJSON.GetAccountResponse.Response.DestinyAccounts[0].UserInfo.MembershipType,
		count, client)

	response.OutputSpeech("All set Guardian.")

	return response, nil
}

func transferItem(itemHash uint, itemSet []*Item, fullCharList []*Character, destCharacter *Character, membershipType uint, count int, client *Client) {

	var totalCount uint
	var wg sync.WaitGroup

	for _, item := range itemSet {

		if item.CharacterIndex != -1 && fullCharList[item.CharacterIndex] == destCharacter {
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
			client.AddAuthHeaders(req)

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
	client.AddAuthHeaders(req)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending transfer request to Bungie API!")
		return
	}

	respBytes, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Response for transfer request: %s\n", string(respBytes))
}

// AllItemsMsg is a type used by channels that need to communicate back from a
// goroutine to the calling function.
type AllItemsMsg struct {
	*ItemsEndpointResponse
	*GetAccountResponse
	error
}

// GetAllItemsForCurrentUser will perform a lookup of the current user based on
// the OAuth credentials provided by Alexa. Then it will make a request to get
// all of the items for that user on all characters.
func GetAllItemsForCurrentUser(client *Client, responseChan chan *AllItemsMsg) {

	start := time.Now()
	currentAccount := GetCurrentAccount(client)
	fmt.Println("Time to get current account: ", time.Since(start))

	if currentAccount == nil {
		fmt.Println("Failed to load current account with the specified access token!")
		responseChan <- &AllItemsMsg{
			ItemsEndpointResponse: nil,
			GetAccountResponse:    nil,
			error:                 errors.New("Couldn't load current user information"),
		}

		return
	}

	startGetItems := time.Now()
	// TODO: Figure out how to support multiple accounts, meaning PSN and XBOX,
	// maybe require it to be specified in the Alexa voice command.
	userInfo := currentAccount.Response.DestinyAccounts[0].UserInfo

	items, err := GetUserItems(userInfo.MembershipType, userInfo.MembershipID, client)
	if err != nil {
		fmt.Println("Failed to read the Items response from Bungie!: ", err.Error())
		responseChan <- &AllItemsMsg{
			ItemsEndpointResponse: nil,
			GetAccountResponse:    currentAccount,
			error:                 errors.New("Failed to read current user's items: " + err.Error()),
		}
		return
	}
	fmt.Println("Time to get user's items: ", time.Since(startGetItems))

	responseChan <- &AllItemsMsg{
		ItemsEndpointResponse: items,
		GetAccountResponse:    currentAccount,
		error:                 nil,
	}
}

// GetCurrentAccount will request the user info for the current user
// based on the OAuth token provided as part of the request.
func GetCurrentAccount(client *Client) *GetAccountResponse {

	req, err := http.NewRequest("GET", GetCurrentAccountEndpoint, nil)
	req.Header.Add("Content-Type", "application/json")
	client.AddAuthHeaders(req)

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
	client.AddAuthHeaders(req)

	startRequest := time.Now()
	itemsResponse, _ := client.Client.Do(req)
	itemsBytes, err := ioutil.ReadAll(itemsResponse.Body)
	fmt.Println("Size of items response: ", len(itemsBytes))
	if err != nil {
		return nil, err
	}
	fmt.Println("Get items request time: ", time.Since(startRequest))

	startUnmarshal := time.Now()
	itemsJSON := &ItemsEndpointResponse{}
	json.Unmarshal(itemsBytes, itemsJSON)
	fmt.Println("Unmarshal items JSON time: ", time.Since(startUnmarshal))

	return itemsJSON, nil
}
