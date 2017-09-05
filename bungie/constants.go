package bungie

// Constant API endpoints
const (
	//GetCurrentAccountEndpoint = "http://localhost:8000/account.json"
	//ItemsEndpointFormat       = "http://localhost:8000/%d/%s/items.json"
	//TransferItemEndpointURL           = "http://localhost:8000/d1/Platform/Destiny/TransferItem/"
	//EquipItemEndpointURL              = "http://localhost:8000/d1/Platform/Destiny/EquipItem/"
	GetCurrentAccountEndpoint         = "https://www.bungie.net/Platform/User/GetCurrentBungieAccount/"
	ItemsEndpointFormat               = "https://www.bungie.net/d1/Platform/Destiny/%d/Account/%s/Items"
	MembershipIDFromDisplayNameFormat = "https://www.bungie.net/d1/Platform/Destiny/SearchDestinyPlayer/%d/%s/"
	TransferItemEndpointURL           = "https://www.bungie.net/d1/Platform/Destiny/TransferItem/"
	EquipItemEndpointURL              = "https://www.bungie.net/d1/Platform/Destiny/EquipItem/"
	TrialsCurrentEndpoint             = "https://api.destinytrialsreport.com/currentMap"
)

// Destiny.TierType
const (
	UnknownTier  = uint(0)
	CurrencyTier = uint(1)
	BasicTier    = uint(2)
	CommonTier   = uint(3)
	RareTier     = uint(4)
	SuperiorTier = uint(5)
	ExoticTier   = uint(6)
)

// Destiny.TansferStatuses
const (
	CanTransfer         = 0
	ItemIsEquipped      = 1
	NotTransferrable    = 2
	NoRoomInDestination = 4
)

// Hash values for different class types 'classHash' JSON key
const (
	WARLOCK = 2271682572
	TITAN   = 3655393761
	HUNTER  = 671679327
)

var classHashToName = map[uint]string{
	WARLOCK: "warlock",
	TITAN:   "titan",
	HUNTER:  "hunter",
}

var classNameToHash = map[string]uint{
	"warlock": WARLOCK,
	"titan":   TITAN,
	"hunter":  HUNTER,
}

// Class Enum value passed in some of the Destiny API responses
const (
	TitanEnum        = uint(0)
	HunterEnum       = uint(1)
	WarlockEnum      = uint(2)
	UnknownClassEnum = uint(3)
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

// BungieMembershipType constant values
const (
	XBOX     = uint(1)
	PSN      = uint(2)
	BLIZZARD = uint(4)
	DEMON    = uint(10)
)

// Alexa doesn't understand some of the dsetiny items or splits them into separate words
// This will allow us to translate to the correct name before doing the lookup.
var commonAlexaItemTranslations = map[string]string{
	"spin metal":     "spinmetal",
	"spin mental":    "spinmetal",
	"passage coins":  "passage coin",
	"strange coins":  "strange coin",
	"exotic shards":  "exotic shard",
	"worm spore":     "wormspore",
	"3 of coins":     "three of coins",
	"worms for":      "wormspore",
	"worm for":       "wormspore",
	"motes":          "mote of light",
	"motes of light": "mote of light",
	"spin middle":    "spinmetal",
}

var commonAlexaClassNameTrnaslations = map[string]string{
	"fault": "vault",
	"tatum": "titan",
}
