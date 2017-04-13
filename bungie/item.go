package bungie

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
