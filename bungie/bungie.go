package bungie

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/mikeflynn/go-alexa/skillserver"
	"github.com/rking788/guardian-helper/db"
)

const (
	// TransferDelay will be the artificial between transfer requests to try and avoid throttling
	TransferDelay = 750 * time.Millisecond
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
		DestinyMemberships []*struct {
			MembershipType uint   `json:"membershipType"`
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

var engramHashes map[uint]bool
var itemMetadata map[uint]*ItemMetadata

// EquipmentBucket is the type of the key for the bucket type hash lookup
type EquipmentBucket uint

func (bucket EquipmentBucket) String() string {
	switch bucket {
	case Primary:
		return "Primary"
	case Special:
		return "Special"
	case Heavy:
		return "Heavy"
	case Ghost:
		return "Ghost"
	case Helmet:
		return "Helmet"
	case Arms:
		return "Arms"
	case Chest:
		return "Chest"
	case Legs:
		return "Legs"
	case ClassArmor:
		return "ClassArmor"
	case Artifact:
		return "Artifact"
	}

	return ""
}

// Equipment bucket type definitions
const (
	Primary EquipmentBucket = iota
	Special
	Heavy
	Ghost
	Helmet
	Arms
	Chest
	Legs
	ClassArmor
	Artifact
)

var bucketHashLookup map[EquipmentBucket]uint

// PopulateEngramHashes will intialize the map holding all item_hash values that represent engram types.
func PopulateEngramHashes() error {

	var err error
	engramHashes, err = db.FindEngramHashes()
	if err != nil {
		fmt.Println("Error populating engram item_hash values: ", err.Error())
		return err
	} else if len(engramHashes) <= 0 {
		fmt.Println("Didn't find any engram item hashes in the database.")
		return errors.New("No engram item_hash values found")
	}

	fmt.Printf("Loaded %d hashes representing engrams into the map.\n", len(engramHashes))
	return nil
}

// PopulateItemMetadata is responsible for loading all of the metadata fields that need
// to be loaded into memory for common inventory related operations.
func PopulateItemMetadata() error {

	rows, err := db.LoadItemMetadata()
	if err != nil {
		return err
	}
	defer rows.Close()

	itemMetadata = make(map[uint]*ItemMetadata)
	for rows.Next() {
		var hash uint
		itemMeta := ItemMetadata{}
		rows.Scan(&hash, &itemMeta.TierType, &itemMeta.ClassType)

		itemMetadata[hash] = &itemMeta
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	fmt.Printf("Loaded %d item metadata entries\n", len(itemMetadata))

	return nil
}

// PopulateBucketHashLookup will fill the map that will be used to lookup bucket type hashes
// which will be used to determine which type of equipment a specific Item represents.
func PopulateBucketHashLookup() error {

	// TODO: This absolutely needs to be done dynamically from the manifest. Not from a static definition
	//var err error
	bucketHashLookup = make(map[EquipmentBucket]uint)

	bucketHashLookup[Primary] = 1498876634
	bucketHashLookup[Special] = 2465295065
	bucketHashLookup[Heavy] = 953998645
	bucketHashLookup[Ghost] = 4023194814

	bucketHashLookup[Helmet] = 3448274439
	bucketHashLookup[Arms] = 3551918588
	bucketHashLookup[Chest] = 14239492
	bucketHashLookup[Legs] = 20886954
	bucketHashLookup[Artifact] = 434908299
	bucketHashLookup[ClassArmor] = 1585787867

	return nil
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

	jsonResponse := MembershipIDLookUpResponse{}
	err = json.NewDecoder(membershipResponse.Body).Decode(&jsonResponse)
	if err != nil {
		fmt.Println("Couldn't read the response body from the Bungie API!")
		return ""
	}

	return jsonResponse.Response[0].MembershipID
}

// CountItem will count the number of the specified item and return an EchoResponse
// that can be serialized and sent back to the Alexa skill.
func CountItem(itemName, accessToken string) (*skillserver.EchoResponse, error) {

	response := skillserver.NewEchoResponse()

	client := NewClient(accessToken, os.Getenv("BUNGIE_API_KEY"))

	// Load all items on all characters
	itemsChannel := make(chan *AllItemsMsg)
	go GetAllItemsForCurrentUser(client, itemsChannel)

	// Check common misinterpretations from Alexa
	if translation, ok := commonAlexaItemTranslations[itemName]; ok {
		itemName = translation
	}

	hash, err := db.GetItemHashFromName(itemName)
	if err != nil {
		outputStr := fmt.Sprintf("Sorry Guardian, I could not find any items named %s in your inventory.", itemName)
		response.OutputSpeech(outputStr)
		return response, nil
	}

	itemsJSON, _ := <-itemsChannel
	if itemsJSON.error != nil {
		response.
			OutputSpeech("Sorry Guardian, I could not load your items from Destiny, you may need to re-link your account in the Alexa app.").
			LinkAccountCard()
		return response, nil
	}
	itemsData := itemsJSON.ItemsEndpointResponse.Response.Data
	matchingItems := itemsData.Items.FilterItems(itemHashFilter, hash)
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

// TransferItem is responsible for calling the necessary Bungie.net APIs to
// transfer the specified item to the specified character. The quantity is optional
// as well as the source class. If no quantity is specified, all of the specific
// items will be transfered to the particular character.
func TransferItem(itemName, accessToken, sourceClass, destinationClass string, count int) (*skillserver.EchoResponse, error) {
	response := skillserver.NewEchoResponse()

	client := NewClient(accessToken, os.Getenv("BUNGIE_API_KEY"))

	itemsChannel := make(chan *AllItemsMsg)
	go GetAllItemsForCurrentUser(client, itemsChannel)

	// Check common misinterpretations from Alexa
	if translation, ok := commonAlexaItemTranslations[itemName]; ok {
		itemName = translation
	}
	if translation, ok := commonAlexaClassNameTrnaslations[destinationClass]; ok {
		destinationClass = translation
	}
	if translation, ok := commonAlexaClassNameTrnaslations[sourceClass]; ok {
		sourceClass = translation
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
	matchingItems := itemsData.Items.FilterItems(itemHashFilter, hash)
	fmt.Printf("Found %d items entries in characters inventory.\n", len(matchingItems))

	if len(matchingItems) == 0 {
		outputStr := fmt.Sprintf("You don't have any %s on any of your characters.", itemName)
		response.OutputSpeech(outputStr)
		return response, nil
	}

	allChars := itemsJSON.ItemsEndpointResponse.Response.Data.Characters
	destCharacter, err := findDestinationCharacter(allChars, destinationClass)
	if err != nil {
		output := fmt.Sprintf("Sorry Guardian, I could not transfer your %s because you do not have any %s characters in Destiny.", itemName, destinationClass)
		fmt.Println(output)
		response.OutputSpeech(output)

		db.InsertUnknownValueIntoTable(destinationClass, db.UnknownClassTable)
		return response, nil
	}

	actualQuantity := transferItem(matchingItems, allChars, destCharacter,
		itemsJSON.GetAccountResponse.Response.DestinyMemberships[0].MembershipType,
		count, client)

	var output string
	if count != -1 && actualQuantity < uint(count) {
		output = fmt.Sprintf("You only had %d %s on other characters, all of it has been transferred to your %s", actualQuantity, itemName, destinationClass)
	} else {
		output = fmt.Sprintf("All set Guardian, %d %s have been transferred to your %s", actualQuantity, itemName, destinationClass)
	}

	response.OutputSpeech(output)

	return response, nil
}

// EquipMaxLightGear will equip all items that are required to have the maximum light on a character
func EquipMaxLightGear(accessToken string) (*skillserver.EchoResponse, error) {
	response := skillserver.NewEchoResponse()

	client := NewClient(accessToken, os.Getenv("BUNGIE_API_KEY"))

	itemsChannel := make(chan *AllItemsMsg)
	go GetAllItemsForCurrentUser(client, itemsChannel)

	itemsJSON := <-itemsChannel
	if itemsJSON.error != nil {
		fmt.Println("Failed to read the Items response from Bungie!: ", itemsJSON.error.Error())
		return nil, itemsJSON.error
	}

	// Transfer to the most recent character on the most recent platform
	destinationIndex := 0
	membershipType := itemsJSON.GetAccountResponse.Response.DestinyMemberships[0].MembershipType

	loadout := findMaxLightLoadout(itemsJSON.ItemsEndpointResponse, destinationIndex)

	fmt.Printf("Found loadout to equip: %v\n", loadout)
	fmt.Printf("Calculated light for loadout: %f\n", loadout.calculateLightLevel())

	err := equipLoadout(loadout, destinationIndex, itemsJSON.ItemsEndpointResponse, membershipType, client)
	if err != nil {
		fmt.Println("Failed to equip the specified loadout: ", err.Error())
		return nil, err
	}

	characterClass := itemsJSON.ItemsEndpointResponse.Response.Data.characterClassNameAtIndex(0)
	response.OutputSpeech(fmt.Sprintf("Max light equipped to your %s Guardian. You are a force to be wreckoned with.", characterClass))
	return response, nil
}

// UnloadEngrams is responsible for transferring all engrams off of a character and
func UnloadEngrams(accessToken string) (*skillserver.EchoResponse, error) {
	response := skillserver.NewEchoResponse()

	client := NewClient(accessToken, os.Getenv("BUNGIE_API_KEY"))

	itemsChannel := make(chan *AllItemsMsg)
	go GetAllItemsForCurrentUser(client, itemsChannel)

	itemsJSON := <-itemsChannel
	if itemsJSON.error != nil {
		fmt.Println("Failed to read the Items response from Bungie!: ", itemsJSON.error.Error())
		return nil, itemsJSON.error
	}

	matchingItems := itemsJSON.ItemsEndpointResponse.Response.Data.Items.FilterItems(itemIsEngramFilter, true)
	if len(matchingItems) == 0 {
		outputStr := fmt.Sprintf("You don't have any engrams on your current character. Happy farming Guardian!")
		response.OutputSpeech(outputStr)
		return response, nil
	}

	foundCount := uint(0)
	for _, item := range matchingItems {
		foundCount += item.Quantity
	}

	fmt.Printf("Found %d engrams on all characters\n", foundCount)

	allChars := itemsJSON.ItemsEndpointResponse.Response.Data.Characters

	_ = transferItem(matchingItems, allChars, nil,
		itemsJSON.GetAccountResponse.Response.DestinyMemberships[0].MembershipType,
		-1, client)

	var output string
	output = fmt.Sprintf("All set Guardian, your engrams have been transferred to your vault. Happy farming Guardian")

	response.OutputSpeech(output)

	return response, nil
}

// GetOutboundIP gets preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

// transferItem is a generic transfer method that will handle a full transfer of a specific item to the specified
// character. This requires a full trip from the source, to the vault, and then to the destination character.
// By providing a nil destCharacter, the items will be transferred to the vault and left there.
func transferItem(itemSet []*Item, fullCharList []*Character, destCharacter *Character, membershipType uint, count int, client *Client) uint {

	// TODO: This should probably take the transferStatus field into account,
	// if the item is NotTransferrable, don't bother trying.
	var totalCount uint
	var wg sync.WaitGroup

	for _, item := range itemSet {

		if item.CharacterIndex != -1 && fullCharList[item.CharacterIndex] == destCharacter {
			continue
		}

		numToTransfer := item.Quantity
		if count != -1 {
			numNeeded := uint(count) - totalCount
			fmt.Printf("Getting to transfer logic: needed=%d, toTransfer=%d\n", numNeeded, numToTransfer)
			if numToTransfer > numNeeded {
				numToTransfer = numNeeded
			}
		}
		totalCount += numToTransfer

		wg.Add(1)

		// TODO: There is an issue were we are getting throttling responses from the Bungie
		// servers. There will be an extra delay added here to try and avoid the throttling.
		go func(item *Item, characters []*Character, wait *sync.WaitGroup) {

			defer wg.Done()

			fmt.Printf("Transferring item: %+v\n", item)

			// If these items are already in the vault, skip it they will be transferred later
			if item.CharacterIndex != -1 {
				// These requests are all going TO the vault, the FROM the vault request
				// will go later for all of these.
				requestBody := map[string]interface{}{
					"itemReferenceHash": item.ItemHash,
					"stackSize":         numToTransfer,
					"transferToVault":   true,
					"itemId":            item.ItemID,
					"characterId":       characters[item.CharacterIndex].CharacterBase.CharacterID,
					"membershipType":    membershipType,
				}

				client.PostTransferItem(requestBody)
				time.Sleep(TransferDelay)
			}

			// TODO: This could possibly be handled more efficiently if we know the items are uniform,
			// meaning they all have the same itemHash values, for example (all motes of light or all strange coins)
			// It is trickier for instances like engrams where each engram type has a different item hash.
			// Now transfer all of these items from the vault to the destination character
			if destCharacter == nil {
				// If the destination is the vault... then we are done already
				return
			}

			vaultToCharRequestBody := map[string]interface{}{
				"itemReferenceHash": item.ItemHash,
				"stackSize":         numToTransfer,
				"transferToVault":   false,
				"itemId":            item.ItemID,
				"characterId":       destCharacter.CharacterBase.CharacterID,
				"membershipType":    membershipType,
			}

			client.PostTransferItem(vaultToCharRequestBody)
			time.Sleep(TransferDelay)

		}(item, fullCharList, &wg)

		if count != -1 && totalCount >= uint(count) {
			break
		}
	}

	wg.Wait()

	return totalCount
}

// equipItems is a generic equip method that will handle a equipping a specific item on a specific character.
func equipItems(itemSet []*Item, characterIndex int, characters []*Character, membershipType uint, client *Client) {

	var wg sync.WaitGroup

	for _, item := range itemSet {

		if item.TransferStatus == ItemIsEquipped && item.CharacterIndex == characterIndex {
			// If this item is already equipped, skip it.
			continue
		}

		wg.Add(1)

		// TODO: There is an issue were we are getting throttling responses from the Bungie
		// servers. There will be an extra delay added here to try and avoid the throttling.
		go func(item *Item, character *Character, membershipType uint, wait *sync.WaitGroup) {

			defer wg.Done()
			equipItem(item, character, membershipType, client)

		}(item, characters[characterIndex], membershipType, &wg)
	}

	wg.Wait()
}

// TODO: All of these equip/transfer/etc. action should take a single struct with all the parameters required
// to perform the action, as well as probably a *Client reference.

// equipItem will take the specified item and equip it on the provided character
func equipItem(item *Item, character *Character, membershipType uint, client *Client) {
	fmt.Printf("Equipping item(%d)...\n", item.ItemHash)

	equipRequestBody := map[string]interface{}{
		"itemId":         item.ItemID,
		"characterId":    character.CharacterBase.CharacterID,
		"membershipType": membershipType,
	}

	client.PostEquipItem(equipRequestBody)
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

	// TODO: check error
	currentAccount, _ := client.GetCurrentAccount()

	if currentAccount == nil {
		fmt.Println("Failed to load current account with the specified access token!")
		responseChan <- &AllItemsMsg{
			ItemsEndpointResponse: nil,
			GetAccountResponse:    nil,
			error:                 errors.New("Couldn't load current user information"),
		}

		return
	}

	// TODO: Figure out how to support multiple accounts, meaning PSN and XBOX,
	// maybe require it to be specified in the Alexa voice command.
	membership := currentAccount.Response.DestinyMemberships[0]

	items, err := client.GetUserItems(membership.MembershipType, membership.MembershipID)
	if err != nil {
		fmt.Println("Failed to read the Items response from Bungie!: ", err.Error())
		responseChan <- &AllItemsMsg{
			ItemsEndpointResponse: nil,
			GetAccountResponse:    currentAccount,
			error:                 errors.New("Failed to read current user's items: " + err.Error()),
		}
		return
	}

	for _, char := range items.Response.Data.Characters {
		fmt.Printf("Found character(%s) with last played date: %+v\n", classHashToName[char.CharacterBase.ClassHash], char.CharacterBase.DateLastPlayed)
	}

	responseChan <- &AllItemsMsg{
		ItemsEndpointResponse: items,
		GetAccountResponse:    currentAccount,
		error:                 nil,
	}
}
