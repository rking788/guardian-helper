package bungie

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

// NOTE: Never run this while using the bungie.net URLs in bungie/constants.go
// those should be changed to a localhost webserver that returns static results.
func BenchmarkSomething(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//CountItem("strange coins", "aaabbbccc")
	}
}

func BenchmarkFiltering(b *testing.B) {
	PopulateItemMetadata()
	items, err := loadItemEndpointResponse()
	if err != nil {
		b.Fail()
		return
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = items.Response.Data.Items.FilterItems(itemTierTypeFilter, ExoticTier)
	}
}

func BenchmarkMaxLight(b *testing.B) {
	PopulateItemMetadata()
	PopulateBucketHashLookup()
	itemsResponse, err := loadItemEndpointResponse()
	if err != nil {
		b.Fail()
		return
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findMaxLightLoadout(itemsResponse, 0)
	}
}

func loadItemEndpointResponse() (*ItemsEndpointResponse, error) {

	f, err := os.Open("../local_tools/samples/get_all_items_summary-latest.json")
	if err != nil {
		fmt.Println("Cannot read local all items response!!: ", err.Error())
		return nil, err
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println("Failed to read local all items response!!: ", err.Error())
		return nil, err
	}

	var response ItemsEndpointResponse
	err = json.Unmarshal(b, &response)
	if err != nil {
		fmt.Println("Error unmarshal-ing the all items response!!: ", err.Error())
		return nil, err
	}

	return &response, nil
}
