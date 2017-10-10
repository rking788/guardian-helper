package bungie

import (
	"fmt"

	"github.com/kpango/glg"
)

//Item represents a single inventory item returned usually by the GetProfile endpoint
type Item struct {
	// DestinyItemComponent https://bungie-net.github.io/multi/schema_Destiny-Entities-Items-DestinyItemComponent.html#schema_Destiny-Entities-Items-DestinyItemComponent
	ItemHash       uint   `json:"itemHash"`
	InstanceID     string `json:"itemInstanceId"`
	BucketHash     uint   `json:"bucketHash"`
	Lockable       bool   `json:"lockable"`
	BindStatus     int    `json:"bindStatus"`
	State          int    `json:"state"`
	Location       int    `json:"location"`
	TransferStatus int    `json:"transferStatus"`
	Quantity       int    `json:"quantity"`
	*ItemInstance
	*Character
}

// ItemInstance will hold information about a specific instance of an instanced item, this can include item stats,
// perks, etc. as well as equipped status and things like that.
type ItemInstance struct {
	//https://bungie-net.github.io/multi/schema_Destiny-Entities-Items-DestinyItemInstanceComponent.html#schema_Destiny-Entities-Items-DestinyItemInstanceComponent
	IsEquipped        bool `json:"isEquipped"`
	CanEquip          bool `json:"canEquip"`
	Quality           int  `json:"quality"`
	CannotEquipReason int  `json:"cannotEquipReason"`
	DamageType        int
	PrimaryStat       *struct {
		//https://bungie-net.github.io/multi/schema_Destiny-DestinyStat.html#schema_Destiny-DestinyStat
		StatHash     uint `json:"statHash"`
		Value        int  `json:"value"`
		MaximumValue int  `json:"maximumValue"`
		ItemLevel    int  `json:"itemLevel"`
	} `json:"primaryStat"`
}

// ItemMetadata is responsible for holding data from the manifest in-memory that is used often
// when interacting wth different character's inventories. These values are used so much
// that it would be a big waste of time to query the manifest data from the DB for every use.
type ItemMetadata struct {
	TierType   int
	ClassType  int
	BucketHash uint
}

func (i *Item) String() string {
	if i.ItemInstance != nil {
		if i.ItemInstance.PrimaryStat != nil {
			return fmt.Sprintf("Item{itemHash: %d, itemID: %s, light:%d, quantity: %d}", i.ItemHash, i.InstanceID, i.PrimaryStat.Value, i.Quantity)
		}

		return fmt.Sprintf("Item{itemHash: %d, itemID: %s, quantity: %d}", i.ItemHash, i.InstanceID, i.Quantity)
	}

	return fmt.Sprintf("Item{itemHash: %d, quantity: %d}", i.ItemHash, i.Quantity)
}

// Power is a convenience accessor to return the power level for a specific item or zero if it does not apply.
func (i *Item) Power() int {
	if i == nil || i.ItemInstance == nil || i.PrimaryStat == nil {
		return 0
	}

	return i.PrimaryStat.Value
}

// IsInVault will determine if the item is in the vault or not. True if it is; False if it is not.
func (i *Item) IsInVault() bool {
	return i.Character == nil
}

// ItemFilter is a type that will be used as a paramter to a filter function.
// The parameter will be a function pointer. The function pointed to will need to return
// true if the element meets some criteria and false otherwise. If the result of
// this filter is false, then the item will be removed.
type ItemFilter func(*Item, interface{}) bool

// ItemList is just a wrapper around a slice of Item pointers. This will make it possible to write a filter
// method that is called on a slice of Items.
type ItemList []*Item

/*
 * Sort Conformance Methods
 */

// LightSort specifies a specific type for ItemList that can be sorted by Light value of each item.
type LightSort ItemList

func (items LightSort) Len() int      { return len(items) }
func (items LightSort) Swap(i, j int) { items[i], items[j] = items[j], items[i] }
func (items LightSort) Less(i, j int) bool {
	return items[i].Power() < items[j].Power()
}

// FilterItems will filter the receiver slice of Items and return only the items that match the criteria
// specified in ItemFilter. If ItemFilter returns True, the element will be included, if it returns False
// the element will be removed.
func (items ItemList) FilterItems(filter ItemFilter, arg interface{}) ItemList {

	result := make(ItemList, 0, len(items))

	for _, item := range items {
		if filter(item, arg) {
			result = append(result, item)
		}
	}

	return result
}

// itemHashFilter will return true if the itemHash provided matches the hash of the item; otherwise false.
func itemHashFilter(item *Item, itemHash interface{}) bool {
	return item != nil && (item.ItemHash == itemHash.(uint))
}

// itemHashesFilter will return true if the item's hash value is present in the provided slice of hashes;
// otherwise false.
func itemHashesFilter(item *Item, hashList interface{}) bool {
	for _, hash := range hashList.([]uint) {
		return itemHashFilter(item, hash)
	}

	return false
}

// itemBucketHashIncludingVaultFilter will filter the list of items by the specified bucket hash or the Vault location
func itemBucketHashFilter(item *Item, bucketTypeHash interface{}) bool {
	metadata, ok := itemMetadata[item.ItemHash]
	if !ok {
		glg.Warnf("No metadata found for item: %s", item.ItemHash)
		return false
	}

	return metadata.BucketHash == bucketTypeHash.(uint)
}

// itemCharacterIDFilter will filter the list of items by the specified character identifier
func itemCharacterIDFilter(item *Item, characterID interface{}) bool {
	return item.Character != nil && (item.Character.CharacterID == characterID.(string))
}

// itemIsEngramFilter will return true if the item represents an engram; otherwise false.
func itemIsEngramFilter(item *Item, wantEngram interface{}) bool {
	_, isEngram := engramHashes[item.ItemHash]
	return isEngram == wantEngram.(bool)
}

// itemTierTypeFilter is a filter that will filter out items that are not of the specified tier.
func itemTierTypeFilter(item *Item, tierType interface{}) bool {
	metadata, ok := itemMetadata[item.ItemHash]
	if !ok {
		glg.Warnf("No metadata found for item: %s", item.ItemHash)
		return false
	}
	return metadata.TierType == tierType.(int)
}

func itemNotTierTypeFilter(item *Item, tierType interface{}) bool {
	metadata, ok := itemMetadata[item.ItemHash]
	if !ok {
		glg.Warnf("No metadata found for item: %s", item.ItemHash)
		return false
	}
	return metadata.TierType != tierType.(int)
}

// itemInstanceIDFilter is an item filter that will return true for all items with an
// instanceID property equal to the one provided. This is useful for filtering a list
// down to a specific instance of an item.
func itemInstanceIDFilter(item *Item, instanceID interface{}) bool {
	return item.InstanceID == instanceID.(string)
}

// itemClassTypeFilter will filter out all items that are not equippable by the specified class
func itemClassTypeFilter(item *Item, classType interface{}) bool {
	// TODO: Is this correct? 3 is UNKNOWN class type, that seems to be what is used for class agnostic items.
	metadata, ok := itemMetadata[item.ItemHash]
	if !ok {
		glg.Warnf("No metadata found for item: %s", item.ItemHash)
		return false
	}

	return (metadata.ClassType == 3) ||
		(metadata.ClassType == classType.(int))
}
