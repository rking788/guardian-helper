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
	_, err := loadItemEndpointResponse()
	if err != nil {
		b.Fail()
		return
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// TODO: Fix this benchmark
		//findMaxLightLoadout(itemsResponse, "")
	}
}

func BenchmarkFixupProfileFromProfileResponse(b *testing.B) {
	response, err := getCurrentProfileResponse()
	if err != nil {
		b.FailNow()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profile := fixupProfileFromProfileResponse(response)
		if profile == nil {
			b.FailNow()
		}
	}
}

func TestParseCurrentMembershipsResponse(t *testing.T) {
	data, err := readSample("GetMembershipsForCurrentUser.json")
	if err != nil {
		fmt.Println("Error reading sample file: ", err.Error())
		t.FailNow()
	}

	var response CurrentUserMembershipsResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		fmt.Println("Error unmarshaling json: ", err.Error())
		t.FailNow()
	}

	if response.Response.BungieNetUser == nil {
		t.FailNow()
	}

	if response.Response.DestinyMemberships == nil {
		t.FailNow()
	}
	if len(response.Response.DestinyMemberships) != 2 {
		t.FailNow()
	}
	for _, membership := range response.Response.DestinyMemberships {
		if membership.DisplayName == "" || membership.MembershipID == "" || membership.MembershipType <= 0 {
			t.FailNow()
		}
		//fmt.Printf("Display name=%s, membershipID=%s, membershipType=%d\n", membership.DisplayName, membership.MembershipID, membership.MembershipType)
	}
}

func TestParseGetProfileResponse(t *testing.T) {

	response, err := getCurrentProfileResponse()
	if err != nil {
		t.FailNow()
	}

	if response.Response.Profile == nil || response.Response.ProfileCurrencies == nil ||
		response.Response.ProfileInventory == nil || response.Response.CharacterEquipment == nil ||
		response.Response.CharacterInventories == nil || response.Response.Characters == nil {
		fmt.Println("One of the expected entries is nil!")
		t.FailNow()
	}

	if len(response.Response.Characters.Data) != 3 {
		t.FailNow()
	}

	if len(response.Response.ProfileCurrencies.Data.Items) != 1 {
		t.FailNow()
	}

	if len(response.Response.CharacterEquipment.Data) == 0 || len(response.Response.CharacterInventories.Data) == 0 {
		t.FailNow()
	}

	for _, char := range response.Response.CharacterEquipment.Data {
		for _, item := range char.Items {
			if item.InstanceID == "" {
				t.FailNow()
			}
		}
	}

	if response.Response.ProfileCurrencies.Data.Items[0].InstanceID != "" {
		t.FailNow()
	}
}

func TestFixupProfileFromProfileResponse(t *testing.T) {

	response, err := getCurrentProfileResponse()
	if err != nil {
		t.FailNow()
	}

	profile := fixupProfileFromProfileResponse(response)
	if profile == nil {
		t.FailNow()
	}

	//fmt.Println("Loaded items: ", profile.AllItems)
}

func getCurrentProfileResponse() (*GetProfileResponse, error) {
	data, err := readSample("GetProfile.json")
	if err != nil {
		fmt.Println("Error reading sample file: ", err.Error())
		return nil, err
	}

	var response GetProfileResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		fmt.Println("Error unmarshaling json: ", err.Error())
		return nil, err
	}

	return &response, nil
}

func readSample(name string) ([]byte, error) {
	f, err := os.Open("../local_tools/samples/" + name)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(f)
}

func loadItemEndpointResponse() (*D1ItemsEndpointResponse, error) {

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

	var response D1ItemsEndpointResponse
	err = json.Unmarshal(b, &response)
	if err != nil {
		fmt.Println("Error unmarshal-ing the all items response!!: ", err.Error())
		return nil, err
	}

	return &response, nil
}
