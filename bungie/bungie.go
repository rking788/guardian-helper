package bungie

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	alexa "github.com/mikeflynn/go-alexa/skillserver"
)

// Constant API endpoints
const (
	AppAuthURL                = "https://www.bungie.net/en/Application/Authorize/2579"
	TokensFromRefreshTokenURL = "https://www.bungie.net/Platform/App/GetAccessTokensFromRefreshToken/"
	TokensFromAuthCodeURL     = "https://www.bungie.net/Platform/App/GetAccessTokensFromCode/"
	ItemsEndpointFormat       = "http://www.bungie.net/Platform/Destiny/%d/Account/%s/Items"
)

// Hash values for different class types 'classHash' JSON key
const (
	WARLOCK = 2271682572
	TITAN   = 3655393761
	HUNTER  = 671679327
)

var classHashToName = map[uint]string{
	WARLOCK: "Warlock",
	TITAN:   "Titan",
	HUNTER:  "Hunter",
}

// Class Enum value passed in some of the Destiny API responses
const (
	TitanEnum        = 0
	HunterEnum       = 1
	WarlockEnum      = 2
	UnknownClassEnum = 3
)

// Hash values for Race types 'raceHash' JSON key
const (
	AWOKEN = 2803282938
	HUMAN  = 3887404748
	EXO    = 898834093
)

// Hash values for Gender 'genderHash' JSON key
const (
	MALE   = 3111576190
	FEMALE = 2204441813
)

// Gender Enum values used in some of the Bungie API responses
const (
	MaleEnum          = 0
	FemaleEnum        = 1
	UnknownGenderEnum = 2
)

// Membership type constant values
const (
	XBOX = 1
	PSN  = 2
)

var itemNameToHashMap = map[string]uint{
	"spin metal":       2882093969,
	"relic iron":       3242866270,
	"spirit bloom":     2254123540,
	"helium filaments": 1797491610,
}

// ItemsEndpointResponse represents the response from a call to the /items endpoint
type ItemsEndpointResponse struct {
	Response        *ItemsResponse `json:"Response"`
	ErrorCode       int            `json:"ErrorCode"`
	ThrottleSeconds int            `json:"ThrottleSeconds"`
	ErrorStatus     string         `json:"ErrorStatus"`
	Message         string         `json:"Message"`
	MessageData     interface{}    `json:"MessageData"`
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
// SAMPLE:
/*
"itemHash": 2547904967,
"itemId": "6917529113043779096",
"quantity": 1,
"damageType": 0,
"damageTypeHash": 0,
"isGridComplete": false,
"transferStatus": 2,
"state": 0,
"characterIndex": 0,
"bucketHash": 1801258597
*/
type Item struct {
	ItemHash       uint   `json:"itemHash"`
	ItemID         string `json:"itemId"`
	Quantity       uint   `json:"quantity"`
	DamageType     uint   `json:"damageType"`
	DamageTypeHash uint   `json:"damageTypeHash"`
	//  IsGridComplete `json:"isGridComplete"`
	TransferStatus uint `json:"transferStatus"`
	State          uint `json:"state"`
	CharacterIndex uint `json:"characterIndex"`
	BucketHash     uint `json:"bucketHash"`
}

// Character will represent a single character entry returned by the /Items endpoint
// SAMPLE:
/*
"characterBase" : ...,
"levelProgression": ...,
"emblemPath": "/common/destiny_content/icons/fb6b9de16fac065c07507569646c5986.jpg",
"backgroundPath": "/common/destiny_content/icons/7e5820dc78c64ce37ee6fc22910ba92a.jpg",
"emblemHash": 2765416092,
"characterLevel": 40,
"baseCharacterLevel": 40,
"isPrestigeLevel": false,
"percentToNextLevel": 0
*/
type Character struct {
	CharacterBase *CharacterBase
	// NOTE: The rest is probably unused at least for the transferring items command
}

// CharacterBase represents the base data for a character entry
// returned by the /Items endpoint.
// SAMPLE:
/*
"membershipId": "4611686018437694484",
"membershipType": 1,
"characterId": "2305843009230596456",
"dateLastPlayed": "2017-03-25T03:44:27Z",
"minutesPlayedThisSession": "114",
"minutesPlayedTotal": "22906",
"powerLevel": 386,
"raceHash": 898834093,
"genderHash": 3111576190,
"classHash": 2271682572,
"currentActivityHash": 0,
"lastCompletedStoryHash": 0,
"stats": {
  "STAT_DEFENSE": {
    "statHash": 3897883278,
    "value": 0,
    "maximumValue": 0
  },
  "STAT_INTELLECT": {
    "statHash": 144602215,
    "value": 274,
    "maximumValue": 0
  },
  "STAT_DISCIPLINE": {
    "statHash": 1735777505,
    "value": 292,
    "maximumValue": 0
  },
  "STAT_STRENGTH": {
    "statHash": 4244567218,
    "value": 92,
    "maximumValue": 0
  },
  "STAT_LIGHT": {
    "statHash": 2391494160,
    "value": 386,
    "maximumValue": 0
  },
  "STAT_ARMOR": {
    "statHash": 392767087,
    "value": 7,
    "maximumValue": 0
  },
  "STAT_AGILITY": {
    "statHash": 2996146975,
    "value": 3,
    "maximumValue": 0
  },
  "STAT_RECOVERY": {
    "statHash": 1943323491,
    "value": 6,
    "maximumValue": 0
  },
  "STAT_OPTICS": {
    "statHash": 3555269338,
    "value": 42,
    "maximumValue": 0
  }
},
"customization": {
  "personality": 2166136261,
  "face": 4017475050,
  "skinColor": 743423469,
  "lipColor": 156633759,
  "eyeColor": 4187018146,
  "hairColor": 1992135330,
  "featureColor": 2166136261,
  "decalColor": 2194048904,
  "wearHelmet": false,
  "hairIndex": 5,
  "featureIndex": 0,
  "decalIndex": 1
},
"grimoireScore": 3855,
"peerView": {
  "equipment": [
    {
      "itemHash": 1256644900,
      "dyes": []
    },
...
  ]
},
"genderType": 0,
"classType": 2,
"buildStatGroupHash": 2257899156
*/
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

// AuthenticationHeaders will generate a map with the required headers to make
// an authenticated HTTP call to the Bungie API.
func AuthenticationHeaders(apiKey, accessToken string) map[string]string {
	return map[string]string{
		"X-Api-Key":     apiKey,
		"Authorization": "Bearer " + accessToken,
	}
}

// TODO: Add a method to retrieve the membership ID from a dispaly name

// MembershipIDFromDisplayName is responsible for retrieving the Destiny
// membership ID from the Bungie API given a specific display name
// from either Xbox or PSN
func MembershipIDFromDisplayName(displayName string) string {
	return "4611686018437694484"
}

// CountItem will count the number of the specified item and return an EchoResponse
// that can be serialized and sent back to the Alexa skill.
func CountItem(itemName, apiKey, accessToken string) (*alexa.EchoResponse, error) {

	response := alexa.NewEchoResponse()

	if _, ok := itemNameToHashMap[itemName]; !ok {
		response.OutputSpeech(fmt.Sprintf("Sorry, I could not find any items named %s in your inventory.", itemName))
		return response, nil
	}

	endpoint := fmt.Sprintf(ItemsEndpointFormat, XBOX, MembershipIDFromDisplayName("rpk788"))

	client := http.Client{}

	req, err := http.NewRequest("GET", endpoint, nil)
	req.Header.Add("Content-Type", "application/json")
	for key, val := range AuthenticationHeaders(apiKey, accessToken) {
		req.Header.Add(key, val)
	}

	itemsResponse, err := client.Do(req)
	itemsBytes, err := ioutil.ReadAll(itemsResponse.Body)
	if err != nil {
		fmt.Println("Failed to read the token response from Bungie!: ", err.Error())
		return nil, err
	}

	itemsJSON := ItemsEndpointResponse{}
	json.Unmarshal(itemsBytes, &itemsJSON)

	fmt.Printf("Unmarshal-ed response from Bungie API: %+v\n", itemsJSON)

	matchingItems := itemsJSON.Response.Data.findItemsMatchingHash(itemNameToHashMap[itemName])
	fmt.Printf("Found %d items in characters inventory.", len(matchingItems))
	fmt.Printf("Matching Items: %+v\n", matchingItems)

	response = response.OutputSpeech("You currently have 12 spinmetal on your Warlock.")

	return response, nil
}

func (data *ItemsData) findItemsMatchingHash(itemHash uint) []*Item {
	result := make([]*Item, 4)

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
