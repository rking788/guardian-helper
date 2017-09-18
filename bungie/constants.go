package bungie

// Constant API endpoints
const (
	//GetCurrentAccountEndpoint = "http://localhost:8000/account.json"
	//ItemsEndpointFormat       = "http://localhost:8000/%d/%s/items.json"
	//TransferItemEndpointURL           = "http://localhost:8000/d1/Platform/Destiny/TransferItem/"
	//EquipItemEndpointURL              = "http://localhost:8000/d1/Platform/Destiny/EquipItem/"
	GetCurrentAccountEndpoint            = "https://www.bungie.net/Platform/User/GetCurrentBungieAccount/"
	GetMembershipsForCurrentUserEndpoint = "https://www.bungie.net/Platform/User/GetMembershipsForCurrentUser/"
	GetProfileEndpointFormat             = "https://www.bungie.net/platform/Destiny2/%d/Profile/%s"
	TransferItemEndpointURL              = "https://www.bungie.net/Platform/Destiny2/Actions/Items/TransferItem/"
	EquipSingleItemEndpointURL           = "https://www.bungie.net/Platform/Destiny2/Actions/Items/EquipItem/"
	EquipMultiItemsEndpointURL           = "https://www.bungie.net/Platform/Destiny2/Actions/Items/EquipItems/"
)

// Component constant values that are needed for certain Bungie API requests that specify which
// collections of values should be returned in the response.
const (
	ProfilesComponent              = "100"
	VendorReceiptsComponent        = "101"
	ProfileInventoriesComponent    = "102"
	ProfileCurrenciesComponent     = "103"
	CharactersComponent            = "200"
	CharacterInventoriesComponent  = "201"
	CharacterProgressionsComponent = "202"
	CharacterRenderDataComponent   = "203"
	CharacterActivitiesComponent   = "204"
	CharacterEquipmentComponent    = "205"
	ItemInstancesComponent         = "300"
	ItemObjectivesComponent        = "301"
	ItemPerksComponent             = "302"
	ItemRenderDataComponent        = "303"
	ItemStatsComponent             = "304"
	ItemSocketsComponent           = "305"
	ItemTalentGridsComponent       = "306"
	ItemCommonDataComponent        = "307"
	ItemPlugStatesComponent        = "308"
	VendorsComponent               = "400"
	VendorCategoriesComponent      = "401"
	VendorSalesComponent           = "402"
	KiosksComponent                = "500"
)

// Destiny.TierType
const (
	UnknownTier  = 0
	CurrencyTier = 1
	BasicTier    = 2
	CommonTier   = 3
	RareTier     = 4
	SuperiorTier = 5
	ExoticTier   = 6
)

// Destiny.TansferStatuses
const (
	CanTransfer         = 0
	ItemIsEquipped      = 1
	NotTransferrable    = 2
	NoRoomInDestination = 4
)

// Destiny.ItemLocation
const (
	UnknownLoc    = 0
	InventoryLoc  = 1
	VaultLoc      = 2
	VendorLoc     = 3
	PostmasterLoc = 4
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
