package bungie

import "fmt"

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
	Items      ItemList      `json:"items"`
	Characters CharacterList `json:"characters"`
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
	PrimaryStat struct {
		StatHash uint `json:"statHash"`
		Value    uint `json:"value"`
		MaxValue uint `json:"maximumValue"`
	} `json:"primaryStat"`
	TransferStatus uint `json:"transferStatus"`
	State          uint `json:"state"`
	CharacterIndex int  `json:"characterIndex"`
	BucketHash     uint `json:"bucketHash"`
}

// ItemMetadata is responsible for holding data from the manifest in-memory that is used often
// when interacting wth different character's inventories. These values are used so much
// that it would be a big waste of time to query the manifest data from the DB for every use.
type ItemMetadata struct {
	TierType  uint
	ClassType uint
}

func (i *Item) String() string {
	return fmt.Sprintf("Item{itemHash: %d, itemID: %s, light:%d, quantity: %d}", i.ItemHash, i.ItemID, i.PrimaryStat.Value, i.Quantity)
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
	return items[i].PrimaryStat.Value < items[j].PrimaryStat.Value
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
	return item.ItemHash == itemHash.(uint)
}

// itemHashesFilter will return true if the item's hash value is present in the provided slice of hashes;
// otherwise false.
func itemHashesFilter(item *Item, hashList interface{}) bool {
	for _, hash := range hashList.([]uint) {
		return itemHashFilter(item, hash)
	}

	return false
}

// itemBucketHashFilter will filter the list of items by the specified bucket hash
func itemBucketHashFilter(item *Item, bucketTypeHash interface{}) bool {
	return item.BucketHash == bucketTypeHash.(uint)
}

// itemCharacterIndexFilter will filter the list of items by the specified character index
func itemCharacterIndexFilter(item *Item, characterIndex interface{}) bool {
	return item.CharacterIndex == characterIndex.(int)
}

// itemIsEngramFilter will return true if the item represents an engram; otherwise false.
func itemIsEngramFilter(item *Item, wantEngram interface{}) bool {
	isEngram := false
	if _, ok := engramHashes[item.ItemHash]; ok {
		isEngram = true
	}

	return isEngram == wantEngram.(bool)
}

// itemTierTypeFilter is a filter that will filter out items that are not of the specified tier.
func itemTierTypeFilter(item *Item, tierType interface{}) bool {
	return itemMetadata[item.ItemHash].TierType == tierType.(uint)
}

func itemNotTierTypeFilter(item *Item, tierType interface{}) bool {
	return itemMetadata[item.ItemHash].TierType != tierType.(uint)
}

// itemClassTypeFilter will filter out all items that are not equippable by the specified class
func itemClassTypeFilter(item *Item, classType interface{}) bool {
	// TODO: Is this correct? 3 is UNKNOWN class type, that seems to be what is used for class agnostic items.
	return (itemMetadata[item.ItemHash].ClassType == 3) ||
		(itemMetadata[item.ItemHash].ClassType == classType.(uint))
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
