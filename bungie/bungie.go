package bungie

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/kpango/glg"
	"github.com/rking788/go-alexa/skillserver"
	"github.com/rking788/guardian-helper/db"
)

const (
	// TransferDelay will be the artificial between transfer requests to try and avoid throttling
	TransferDelay = 750 * time.Millisecond
)

// Equipment bucket type definitions
const (
	_ EquipmentBucket = iota
	Kinetic
	Energy
	Power
	Ghost
	Helmet
	Gauntlets
	Chest
	Legs
	ClassArmor
	Artifact
)

// Clients stores a list of bungie.Client instances that can be used to make HTTP requests to the Bungie API
var Clients *ClientPool

// It is probably faster to just load all of the item_name->item_hash lookups into memory.
// That way we can give feedback to the user quicker if an item name is not found.
// If memory overhead becomes an issue this can be removed and go back to the DB lookups.
var itemHashLookup map[string]uint

var engramHashes map[uint]bool
var itemMetadata map[uint]*ItemMetadata
var bucketHashLookup map[EquipmentBucket]uint
var equipmentBucketLookup map[uint]EquipmentBucket
var bungieAPIKey string

// InitEnv provides a package level initialization point for any work that is environment specific
func InitEnv(apiKey string) {
	bungieAPIKey = apiKey

	Clients = NewClientPool()

	err := PopulateEngramHashes()
	if err != nil {
		glg.Errorf("Error populating engram hashes: %s\nExiting...", err.Error())
		return
	}
	err = PopulateBucketHashLookup()
	if err != nil {
		glg.Errorf("Error populating bucket hash values: %s\nExiting...", err.Error())
		return
	}
	err = PopulateItemMetadata()
	if err != nil {
		glg.Errorf("Error populating item metadata lookup table: %s\nExiting...", err.Error())
		return
	}
}

// EquipmentBucket is the type of the key for the bucket type hash lookup
type EquipmentBucket uint

func (bucket EquipmentBucket) String() string {
	switch bucket {
	case Kinetic:
		return "Kinetic"
	case Energy:
		return "Energy"
	case Power:
		return "Power"
	case Ghost:
		return "Ghost"
	case Helmet:
		return "Helmet"
	case Gauntlets:
		return "Gauntlets"
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

// PopulateEngramHashes will intialize the map holding all item_hash values that represent engram types.
func PopulateEngramHashes() error {

	var err error
	engramHashes, err = db.FindEngramHashes()
	if err != nil {
		glg.Errorf("Error populating engram item_hash values: %s", err.Error())
		return err
	} else if len(engramHashes) <= 0 {
		glg.Error("Didn't find any engram item hashes in the database.")
		return errors.New("No engram item_hash values found")
	}

	glg.Infof("Loaded %d hashes representing engrams into the map.", len(engramHashes))
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
	itemHashLookup = make(map[string]uint)
	for rows.Next() {
		var hash uint
		var itemName string
		itemMeta := ItemMetadata{}
		rows.Scan(&hash, &itemName, &itemMeta.TierType, &itemMeta.ClassType, &itemMeta.BucketHash)

		itemMetadata[hash] = &itemMeta
		if itemName != "" {
			itemHashLookup[itemName] = hash
		} else {
			glg.Warn("Found an empty item name, skipping...")
		}
	}
	if rows.Err() != nil {
		return rows.Err()
	}
	glg.Infof("Loaded %d item metadata entries", len(itemMetadata))

	return nil
}

// PopulateBucketHashLookup will fill the map that will be used to lookup bucket type hashes
// which will be used to determine which type of equipment a specific Item represents.
func PopulateBucketHashLookup() error {

	// TODO: This absolutely needs to be done dynamically from the manifest. Not from a static definition
	//var err error
	bucketHashLookup = make(map[EquipmentBucket]uint)

	bucketHashLookup[Kinetic] = 1498876634
	bucketHashLookup[Energy] = 2465295065
	bucketHashLookup[Power] = 953998645
	bucketHashLookup[Ghost] = 4023194814

	bucketHashLookup[Helmet] = 3448274439
	bucketHashLookup[Gauntlets] = 3551918588
	bucketHashLookup[Chest] = 14239492
	bucketHashLookup[Legs] = 20886954
	bucketHashLookup[Artifact] = 434908299
	bucketHashLookup[ClassArmor] = 1585787867

	equipmentBucketLookup = make(map[uint]EquipmentBucket)
	equipmentBucketLookup[1498876634] = Kinetic
	equipmentBucketLookup[2465295065] = Energy
	equipmentBucketLookup[953998645] = Power
	equipmentBucketLookup[4023194814] = Ghost

	equipmentBucketLookup[3448274439] = Helmet
	equipmentBucketLookup[3551918588] = Gauntlets
	equipmentBucketLookup[14239492] = Chest
	equipmentBucketLookup[20886954] = Legs
	equipmentBucketLookup[434908299] = Artifact
	equipmentBucketLookup[1585787867] = ClassArmor

	return nil
}

// CountItem will count the number of the specified item and return an EchoResponse
// that can be serialized and sent back to the Alexa skill.
func CountItem(itemName, accessToken string) (*skillserver.EchoResponse, error) {
	glg.Infof("ItemName: %s", itemName)

	response := skillserver.NewEchoResponse()

	// Check common misinterpretations from Alexa
	if translation, ok := commonAlexaItemTranslations[itemName]; ok {
		itemName = translation
	}

	// hash, err := db.GetItemHashFromName(itemName)
	// if err != nil {
	hash, ok := itemHashLookup[itemName]
	if !ok {
		outputStr := fmt.Sprintf("Sorry Guardian, I could not find any items named %s in your inventory.", itemName)
		response.OutputSpeech(outputStr)
		return response, nil
	}

	client := Clients.Get()
	client.AddAuthValues(accessToken, bungieAPIKey)

	// Load all items on all characters
	profileChannel := make(chan *ProfileMsg)
	go GetProfileForCurrentUser(client, profileChannel)

	msg, _ := <-profileChannel
	if msg.error != nil {
		response.
			OutputSpeech("Sorry Guardian, I could not load your items from Destiny, you may need to re-link your account in the Alexa app.").
			LinkAccountCard()
		return response, nil
	}
	matchingItems := msg.Profile.AllItems.FilterItems(itemHashFilter, hash)
	glg.Infof("Found %d items entries in characters inventory.", len(matchingItems))

	if len(matchingItems) == 0 {
		outputStr := fmt.Sprintf("You don't have any %s on any of your characters.", itemName)
		response.OutputSpeech(outputStr)
		return response, nil
	}

	outputString := ""
	for _, item := range matchingItems {
		if item.Character == nil {
			outputString += fmt.Sprintf("You have %d %s on your account", item.Quantity, itemName)
		} else {
			outputString += fmt.Sprintf("Your %s has %d %s. ", classHashToName[item.Character.ClassHash], item.Quantity, itemName)
		}
	}
	response = response.OutputSpeech(outputString)

	return response, nil
}

// TransferItem is responsible for calling the necessary Bungie.net APIs to
// transfer the specified item to the specified character. The quantity is optional
// as well as the source class. If no quantity is specified, all of the specific
// items will be transfered to the particular character.
func TransferItem(itemName, accessToken, sourceClass, destinationClass string, count int) (*skillserver.EchoResponse, error) {
	glg.Infof("ItemName: %s, Source: %s, Destination: %s, Count: %d", itemName, sourceClass, destinationClass, count)

	response := skillserver.NewEchoResponse()

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

	//hash, err := db.GetItemHashFromName(itemName)
	//if err != nil {
	hash, ok := itemHashLookup[itemName]
	if !ok {
		outputStr := fmt.Sprintf("Sorry Guardian, I could not find any items named %s in your inventory.", itemName)
		response.OutputSpeech(outputStr)
		return response, nil
	}

	client := Clients.Get()
	client.AddAuthValues(accessToken, bungieAPIKey)

	profileChannel := make(chan *ProfileMsg)
	go GetProfileForCurrentUser(client, profileChannel)

	msg := <-profileChannel
	if msg.error != nil {
		glg.Errorf("Failed to read the Items response from Bungie!: %s", msg.error.Error())
		return nil, msg.error
	}

	matchingItems := msg.Profile.AllItems.FilterItems(itemHashFilter, hash)
	glg.Infof("Found %d items entries in characters inventory.", len(matchingItems))

	if len(matchingItems) == 0 {
		outputStr := fmt.Sprintf("You don't have any %s on any of your characters.", itemName)
		response.OutputSpeech(outputStr)
		return response, nil
	}

	allChars := msg.Profile.Characters
	destCharacter, err := allChars.findDestinationCharacter(destinationClass)
	if err != nil {
		output := fmt.Sprintf("Sorry Guardian, I could not transfer your %s because you do not have any %s characters in Destiny.", itemName, destinationClass)
		glg.Error(output)
		response.OutputSpeech(output)

		db.InsertUnknownValueIntoTable(destinationClass, db.UnknownClassTable)
		return response, nil
	}

	actualQuantity := transferItem(matchingItems, allChars, destCharacter,
		msg.Profile.MembershipType, count, client)

	var output string
	if count != -1 && actualQuantity < count {
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

	client := Clients.Get()
	client.AddAuthValues(accessToken, bungieAPIKey)

	profileChannel := make(chan *ProfileMsg)
	go GetProfileForCurrentUser(client, profileChannel)

	msg := <-profileChannel
	if msg.error != nil {
		glg.Errorf("Failed to read the Items response from Bungie!: %s", msg.error.Error())
		return nil, msg.error
	}

	// Transfer to the most recent character on the most recent platform
	destinationID := msg.Profile.Characters[0].CharacterID
	membershipType := msg.Profile.MembershipType

	glg.Debugf("Character(%s), MembershipID(%s), MembershipType(%d)",
		msg.Profile.Characters[0].CharacterID, msg.Profile.MembershipID, msg.Profile.MembershipType)

	loadout := findMaxLightLoadout(msg.Profile, destinationID)

	glg.Debugf("Found loadout to equip: %v", loadout)
	glg.Infof("Calculated light for loadout: %f", loadout.calculateLightLevel())

	err := equipLoadout(loadout, destinationID, msg.Profile, membershipType, client)
	if err != nil {
		glg.Errorf("Failed to equip the specified loadout: %s", err.Error())
		return nil, err
	}

	characterClass := classHashToName[msg.Profile.Characters[0].ClassHash]
	response.OutputSpeech(fmt.Sprintf("Max light equipped to your %s Guardian. You are a force to be wreckoned with.", characterClass))
	return response, nil
}

// UnloadEngrams is responsible for transferring all engrams off of a character and
func UnloadEngrams(accessToken string) (*skillserver.EchoResponse, error) {
	response := skillserver.NewEchoResponse()

	client := Clients.Get()
	client.AddAuthValues(accessToken, bungieAPIKey)

	profileChannel := make(chan *ProfileMsg)
	go GetProfileForCurrentUser(client, profileChannel)

	msg := <-profileChannel
	if msg.error != nil {
		glg.Errorf("Failed to read the Items response from Bungie!: %s", msg.error.Error())
		return nil, msg.error
	}

	matchingItems := msg.Profile.AllItems.FilterItems(itemIsEngramFilter, true)
	if len(matchingItems) == 0 {
		outputStr := fmt.Sprintf("You don't have any engrams on your current character. Happy farming Guardian!")
		response.OutputSpeech(outputStr)
		return response, nil
	}

	foundCount := 0
	for _, item := range matchingItems {
		foundCount += item.Quantity
	}

	glg.Infof("Found %d engrams on all characters", foundCount)

	allChars := msg.Profile.Characters

	_ = transferItem(matchingItems, allChars, nil,
		msg.Profile.MembershipType, -1, client)

	var output string
	output = fmt.Sprintf("All set Guardian, your engrams have been transferred to your vault. Happy farming Guardian")

	response.OutputSpeech(output)

	return response, nil
}

// CreateLoadoutForCurrentCharacter will create a new PersistedLoadout based on the items equipped
// to the user's current character and save them to the persistent storage.
func CreateLoadoutForCurrentCharacter(accessToken, name string, shouldOverwrite bool) (*skillserver.EchoResponse, error) {

	response := skillserver.NewEchoResponse()

	glg.Infof("Creating new loadout named: %s", name)
	if name == "" {
		response.OutputSpeech("Sorry Guardian, you need to provide a loadout name based on an " +
			"activtiy like crucible, strikes, or patrols.")
		return response, nil
	}

	client := Clients.Get()
	client.AddAuthValues(accessToken, bungieAPIKey)

	// TODO: check error
	currentAccount, _ := client.GetCurrentAccount()

	if currentAccount == nil {
		glg.Error("Failed to load current account with the specified access token!")
		return nil, errors.New("Couldn't load the current account")
	}

	// Check to see if a loadout with this name already exists and prompt for
	// confirmation to overwrite
	bnetMembershipID := currentAccount.Response.BungieNetUser.MembershipID
	if !shouldOverwrite {
		existing, _ := db.SelectLoadout(bnetMembershipID, name)
		if existing != "" {
			// Prompt the user to see if they want to overwrite the existing loadout
			response.ConfirmIntent("CreateLoadout", nil).
				OutputSpeech(fmt.Sprintf("You already have a loadout named %s, would you like to overwrite it?", name))
			return response, nil
		}
	}

	// TODO: Figure out how to support multiple accounts, meaning PSN and XBOX,
	// maybe require it to be specified in the Alexa voice command.
	membership := currentAccount.Response.DestinyMemberships[0]

	profileResponse, err := client.GetCurrentEquipment(membership.MembershipType,
		membership.MembershipID)
	if err != nil {
		glg.Errorf("Failed to read the Profile response from Bungie!: %s", err.Error())
		return nil, errors.New("Failed to read current user's profile: " + err.Error())
	}

	profile := fixupProfileFromProfileResponse(profileResponse)
	profile.BungieNetMembershipID = bnetMembershipID

	// We want to remove all items that are not on the current character
	profile.AllItems = profile.AllItems.FilterItems(itemCharacterIDFilter,
		profile.Characters[0].CharacterID)

	loadout := loadoutFromProfile(profile)
	glg.Debugf("Created Loadout: %+v", loadout)
	persistedLoadout := loadout.toPersistedLoadout()
	persistedBytes, err := json.Marshal(persistedLoadout)
	if err != nil {
		glg.Errorf("Failed to marshal the loadout to JSON: %s", err.Error())
		return nil, err
	}

	// TODO: This should handle the case where a loadout already exists with this name
	if shouldOverwrite {
		db.UpdateLoadout(persistedBytes, bnetMembershipID, name)
	} else {
		db.SaveLoadout(persistedBytes, bnetMembershipID, name)
	}

	response.OutputSpeech("All set Guardian, your " + name + " loadout was saved for you.")

	return response, nil
}

func EquipNamedLoadout(accessToken, name string) (*skillserver.EchoResponse, error) {

	response := skillserver.NewEchoResponse()

	client := Clients.Get()
	client.AddAuthValues(accessToken, bungieAPIKey)

	// TODO: check error
	currentAccount, _ := client.GetCurrentAccount()

	if currentAccount == nil {
		glg.Error("Failed to load current account with the specified access token!")
		return nil, errors.New("CLouldn't load the current account")
	}

	// TODO: Figure out how to support multiple accounts, meaning PSN and XBOX,
	// maybe require it to be specified in the Alexa voice command.
	membership := currentAccount.Response.DestinyMemberships[0]

	profileResponse, err := client.GetUserProfileData(membership.MembershipType,
		membership.MembershipID)
	if err != nil {
		glg.Errorf("Failed to read the Profile response from Bungie!: %s", err.Error())
		return nil, errors.New("Failed to read current user's profile: " + err.Error())
	}

	profile := fixupProfileFromProfileResponse(profileResponse)
	profile.BungieNetMembershipID = currentAccount.Response.BungieNetUser.MembershipID

	loadoutJSON, err := db.SelectLoadout(profile.BungieNetMembershipID, name)
	if err == nil && loadoutJSON == "" {
		response.OutputSpeech("Sorry Guardian, a loadout could not be found with the name " + name)
		return response, nil
	} else if err != nil {
		glg.Errorf("Failed to read loadout from the database")
		return nil, err
	}

	var peristedLoadout PersistedLoadout
	err = json.NewDecoder(bytes.NewReader([]byte(loadoutJSON))).Decode(&peristedLoadout)
	if err != nil {
		glg.Errorf("Failed to decode JSON: %s", err.Error())
		return nil, err
	}

	loadout := fromPersistedLoadout(peristedLoadout, profile)
	equipLoadout(loadout, profile.Characters[0].CharacterID, profile,
		profile.MembershipType, client)

	response.OutputSpeech("All set Guardian, your " + name + " loadout has been restored!")

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

// transferItem is a generic transfer method that will handle a full transfer of a specific item to
// the specified character. This requires a full trip from the source, to the vault, and then to the
// destination character. By providing a nil destCharacter, the items will be transferred to the
// vault and left there.
func transferItem(itemSet []*Item, fullCharList []*Character, destCharacter *Character,
	membershipType int, count int, client *Client) int {

	// TODO: This should probably take the transferStatus field into account,
	// if the item is NotTransferrable, don't bother trying.
	var totalCount int
	var wg sync.WaitGroup

	for _, item := range itemSet {

		if item.Character == destCharacter {
			// Item is already on the destination character, skipping...
			continue
		}

		numToTransfer := item.Quantity
		if count != -1 {
			numNeeded := count - totalCount
			glg.Debugf("Getting to transfer logic: needed=%d, toTransfer=%d", numNeeded, numToTransfer)
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

			glg.Infof("Transferring item: %+v", item)

			// TODO: I think this is probably the best way to tell if it is in the vault?
			// maybe need to check the bucket hash instead?

			// If these items are already in the vault, skip it they will be transferred later
			if item.Character != nil {
				// These requests are all going TO the vault, the FROM the vault request
				// will go later for all of these.
				requestBody := map[string]interface{}{
					"itemReferenceHash": item.ItemHash,
					"stackSize":         numToTransfer,
					"transferToVault":   true,
					"itemId":            item.InstanceID,
					"characterId":       item.Character.CharacterID,
					"membershipType":    membershipType,
				}

				transferClient := Clients.Get()
				transferClient.AddAuthValues(client.AccessToken, client.APIToken)
				transferClient.PostTransferItem(requestBody)
			}

			// TODO: This could possibly be handled more efficiently if we know the items are
			// uniform, meaning they all have the same itemHash values, for example (all motes of
			// light or all strange coins) It is trickier for instances like engrams where each
			// engram type has a different item hash. Now transfer all of these items from the
			// vault to the destination character
			if destCharacter == nil {
				// If the destination is the vault... then we are done already
				return
			}

			vaultToCharRequestBody := map[string]interface{}{
				"itemReferenceHash": item.ItemHash,
				"stackSize":         numToTransfer,
				"transferToVault":   false,
				"itemId":            item.InstanceID,
				"characterId":       destCharacter.CharacterID,
				"membershipType":    membershipType,
			}

			transferClient := Clients.Get()
			transferClient.AddAuthValues(client.AccessToken, client.APIToken)
			transferClient.PostTransferItem(vaultToCharRequestBody)

		}(item, fullCharList, &wg)

		if count != -1 && totalCount >= count {
			break
		}
	}

	wg.Wait()

	return totalCount
}

// equipItems is a generic equip method that will handle a equipping a specific
// item on a specific character.
func equipItems(itemSet []*Item, characterID string, characters CharacterList,
	membershipType int, client *Client) {

	ids := make([]int64, 0, len(itemSet))

	for _, item := range itemSet {

		if item.TransferStatus == ItemIsEquipped && item.Character.CharacterID == characterID {
			// If this item is already equipped, skip it.
			glg.Debugf("Not equipping item because it is already equipped on the current character: %s", item.InstanceID)
			continue
		}

		instanceID, err := strconv.ParseInt(item.InstanceID, 10, 64)
		if err != nil {
			glg.Errorf("Not equipping item because the instance ID could not be parsed to an Int: %s", err.Error())
			continue
		}
		ids = append(ids, instanceID)
	}

	glg.Debugf("Equipping items: %+v", ids)

	equipRequestBody := map[string]interface{}{
		"itemIds":        ids,
		"characterId":    characterID,
		"membershipType": membershipType,
	}

	// Having a single equip call should avoid the throttling problems.
	client.PostEquipItem(equipRequestBody, true)
}

// TODO: All of these equip/transfer/etc. action should take a single struct with all the
// parameters required to perform the action, as well as probably a *Client reference.

// equipItem will take the specified item and equip it on the provided character
func equipItem(item *Item, character *Character, membershipType int, client *Client) {
	glg.Debugf("Equipping item(%d, %d)...", item.ItemHash, item.InstanceID)

	equipRequestBody := map[string]interface{}{
		"itemId":         item.InstanceID,
		"characterId":    character.CharacterID,
		"membershipType": membershipType,
	}

	client.PostEquipItem(equipRequestBody, false)
}

// Profile contains all information about a specific Destiny membership, including character and
// inventory information.
type Profile struct {
	MembershipType        int
	MembershipID          string
	BungieNetMembershipID string
	DateLastPlayed        time.Time
	DisplayName           string
	Characters            CharacterList

	AllItems ItemList
	// NOTE: Still not sure this is the best approach to flatten items into a single list,
	// it works well for now so we will go with it. There are too many potential spots to look for an item.
	//Equipments       map[string]ItemList
	//Inventories      map[string]ItemList
	//ProfileInventory ItemList
	//Currencies       ItemList
}

// ProfileMsg is a wrapper around a Profile struct that should be used exclusively for sending a
// Profile over a channel, or at least in cases where an error also needs to be sent to indicate
// failures.
type ProfileMsg struct {
	*Profile
	error
}

// GetProfileForCurrentUser will retrieve the Profile data for the currently logged in user
// (determined by the access_token)
func GetProfileForCurrentUser(client *Client, responseChan chan *ProfileMsg) {

	// TODO: check error
	currentAccount, _ := client.GetCurrentAccount()

	if currentAccount == nil {
		glg.Error("Failed to load current account with the specified access token!")
		responseChan <- &ProfileMsg{
			Profile: nil,
			error:   errors.New("Couldn't load current user information"),
		}

		return
	}

	// TODO: Figure out how to support multiple accounts, meaning PSN and XBOX,
	// maybe require it to be specified in the Alexa voice command.
	membership := currentAccount.Response.DestinyMemberships[0]

	profileResponse, err := client.GetUserProfileData(membership.MembershipType, membership.MembershipID)
	if err != nil {
		glg.Errorf("Failed to read the Profile response from Bungie!: %s", err.Error())
		responseChan <- &ProfileMsg{
			Profile: nil,
			error:   errors.New("Failed to read current user's profile: " + err.Error()),
		}
		return
	}

	profile := fixupProfileFromProfileResponse(profileResponse)
	profile.BungieNetMembershipID = currentAccount.Response.BungieNetUser.MembershipID

	for _, char := range profile.Characters {
		glg.Debugf("Character(%s) last played date: %+v", classHashToName[char.ClassHash], char.DateLastPlayed)
	}

	responseChan <- &ProfileMsg{
		Profile: profile,
		error:   nil,
	}
}

func loadoutFromProfile(profile *Profile) Loadout {
	loadout := make(Loadout)
	for _, item := range profile.AllItems {
		glg.Debugf("Found item(%d) for bucket(%d), equipment bucket lookupresult(%d)", item.ItemHash, item.BucketHash, equipmentBucketLookup[item.BucketHash])
		if equipmentBucket, ok := equipmentBucketLookup[item.BucketHash]; ok {
			if _, ok := loadout[equipmentBucket]; ok {
				glg.Debugf("Found duplicate item for bucket: %d", item.BucketHash)
			}
			loadout[equipmentBucket] = item
		}
	}

	return loadout
}

func fixupProfileFromProfileResponse(response *GetProfileResponse) *Profile {
	profile := &Profile{}

	// Profile Component
	if response.Response.Profile != nil {
		profile.MembershipID = response.Response.Profile.Data.UserInfo.MembershipID
		profile.MembershipType = response.Response.Profile.Data.UserInfo.MembershipType
	}

	// Transform character map into an ordered list based on played time.
	// Characters Component
	if response.Response.Characters != nil {
		profile.Characters = make([]*Character, 0, len(response.Response.Characters.Data))
		for _, char := range response.Response.Characters.Data {
			profile.Characters = append(profile.Characters, char)
		}

		sort.Sort(sort.Reverse(LastPlayedSort(profile.Characters)))
	}

	// Flatten out the items from different buckets including currencies, inventories, eequipments,
	// etc.
	//totalItemCount := len(response.Response.ProfileCurrencies.Data.Items) + len(response.Response.ProfileInventory.Data.Items)
	// for id := range response.Response.Characters.Data {
	// 	totalItemCount += len(response.Response.CharacterEquipment.Data[id].Items)
	// 	totalItemCount += len(response.Response.CharacterInventories.Data[id].Items)
	// }

	items := make(ItemList, 0, 32)

	// ProfileCurrencies Component
	if response.Response.ProfileCurrencies != nil {
		items = append(items, response.Response.ProfileCurrencies.Data.Items...)
	}

	// ProfileInventory Component
	if response.Response.ProfileInventory != nil {
		for _, item := range response.Response.ProfileInventory.Data.Items {
			if item.InstanceID != "" {
				item.ItemInstance = response.Response.ItemComponents.Instances.Data[item.InstanceID]
			}
		}
		items = append(items, response.Response.ProfileInventory.Data.Items...)
	}

	// CharacterEquipment Component
	if response.Response.CharacterEquipment != nil {
		for charID, list := range response.Response.CharacterEquipment.Data {
			for _, item := range list.Items {
				if response.Response.Characters != nil {
					item.Character = response.Response.Characters.Data[charID]
				}
				if item.InstanceID != "" && response.Response.ItemComponents != nil &&
					response.Response.ItemComponents.Instances != nil {
					item.ItemInstance = response.Response.ItemComponents.Instances.Data[item.InstanceID]
				}
			}

			items = append(items, list.Items...)
		}
	}

	// CharacterInventories Component
	if response.Response.CharacterInventories != nil {
		for charID, list := range response.Response.CharacterInventories.Data {
			for _, item := range list.Items {
				if response.Response.Characters != nil {
					item.Character = response.Response.Characters.Data[charID]
				}
				if item.InstanceID != "" && response.Response.ItemComponents != nil &&
					response.Response.ItemComponents.Instances != nil {
					item.ItemInstance = response.Response.ItemComponents.Instances.Data[item.InstanceID]
				}
			}
			items = append(items, list.Items...)
		}
	}

	profile.AllItems = items

	return profile
}
