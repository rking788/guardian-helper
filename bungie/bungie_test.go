package bungie

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/kpango/glg"

	"github.com/rking788/guardian-helper/db"
)

func setup() {
	glg.Get().SetLevelMode(glg.DEBG, glg.NONE)
	glg.Get().SetLevelMode(glg.INFO, glg.NONE)
	glg.Get().SetLevelMode(glg.WARN, glg.NONE)

	db.InitEnv(os.Getenv("DATABASE_URL"))
	InitEnv("")
}

// NOTE: Never run this while using the bungie.net URLs in bungie/constants.go
// those should be changed to a localhost webserver that returns static results.
// func BenchmarkSomething(b *testing.B) {

// 	profileResponse, err := getCurrentProfileResponse()
// 	if err != nil {
// 		b.Fail()
// 		return
// 	}
// 	_ = fixupProfileFromProfileResponse(profileResponse)

// 	b.ReportAllocs()
// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		//CountItem("strange coins", "aaabbbccc")
// 	}
// }

func BenchmarkFiltering(b *testing.B) {
	setup()
	profileResponse, err := getCurrentProfileResponse()
	if err != nil {
		b.FailNow()
		return
	}
	profile := fixupProfileFromProfileResponse(profileResponse)

	items := profile.AllItems
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = items.FilterItems(itemTierTypeFilter, ExoticTier)
	}
}

func BenchmarkMaxLight(b *testing.B) {
	setup()
	profileResponse, err := getCurrentProfileResponse()
	if err != nil {
		b.FailNow()
		return
	}
	profile := fixupProfileFromProfileResponse(profileResponse)
	testDestinationID := profile.Characters[0].CharacterID

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findMaxLightLoadout(profile, testDestinationID)
	}
}

func BenchmarkGroupAndSort(b *testing.B) {
	setup()
	response, err := getCurrentProfileResponse()
	if err != nil {
		b.FailNow()
	}
	profile := fixupProfileFromProfileResponse(response)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		profile := groupAndSortGear(profile.AllItems)
		if profile == nil {
			b.FailNow()
		}
	}
}

func BenchmarkBestItemForBucket(b *testing.B) {
	setup()
	response, err := getCurrentProfileResponse()
	if err != nil {
		b.FailNow()
	}
	profile := fixupProfileFromProfileResponse(response)
	grouped := groupAndSortGear(profile.AllItems)
	largestBucket := Kinetic
	largestBucketSize := len(grouped[Kinetic])
	for bkt, list := range grouped {
		if len(list) > largestBucketSize {
			largestBucket = bkt
			largestBucketSize = len(list)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := findBestItemForBucket(largestBucket, grouped[largestBucket], profile.Characters[0].CharacterID)
		if item == nil {
			b.FailNow()
		}
	}
}

func BenchmarkFixupProfileFromProfileResponse(b *testing.B) {
	setup()
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

func BenchmarkLoadoutFromProfile(b *testing.B) {
	setup()

	setup()
	response, err := getCurrentProfileResponse()
	if err != nil {
		b.FailNow()
	}
	response.Response.CharacterInventories = nil
	response.Response.ProfileInventory = nil
	response.Response.ProfileCurrencies = nil
	response.Response.Profile = nil
	response.Response.ItemComponents = nil

	profile := fixupProfileFromProfileResponse(response)
	if profile == nil {
		b.FailNow()
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loadout := loadoutFromProfile(profile)
		if loadout == nil {
			b.FailNow()
		}
	}
}

func TestParseCurrentMembershipsResponse(t *testing.T) {
	setup()
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
	setup()
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

	setup()
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

func TestFixupProfileFromProfileResponseMissingProfile(t *testing.T) {

	setup()
	response, err := getCurrentProfileResponse()
	if err != nil {
		t.FailNow()
	}
	response.Response.Profile = nil

	profile := fixupProfileFromProfileResponse(response)
	if profile == nil {
		t.FailNow()
	}

	if profile.MembershipID != "" {
		t.FailNow()
	}
	if profile.MembershipType != 0 {
		t.FailNow()
	}

	//fmt.Println("Loaded items: ", profile.AllItems)
}

func TestFixupProfileFromProfileResponseMissingProfileInventory(t *testing.T) {

	setup()
	response, err := getCurrentProfileResponse()
	if err != nil {
		t.FailNow()
	}
	response.Response.ProfileInventory = nil

	profile := fixupProfileFromProfileResponse(response)
	if profile == nil {
		t.FailNow()
	}
}

func TestFixupProfileFromProfileResponseMissingCharacters(t *testing.T) {

	setup()
	response, err := getCurrentProfileResponse()
	if err != nil {
		t.FailNow()
	}
	response.Response.Characters = nil

	profile := fixupProfileFromProfileResponse(response)
	if profile == nil {
		t.FailNow()
	}

	if profile.Characters != nil {
		t.FailNow()
	}

	for _, item := range profile.AllItems {
		if item.Character != nil {
			t.FailNow()
		}
	}
}

func TestFixupProfileFromProfileResponseMissingCharacterEquipment(t *testing.T) {

	setup()
	response, err := getCurrentProfileResponse()
	if err != nil {
		t.FailNow()
	}
	response.Response.CharacterEquipment = nil

	profile := fixupProfileFromProfileResponse(response)
	if profile == nil {
		t.FailNow()
	}

	for _, item := range profile.AllItems {
		if item.ItemInstance != nil && item.IsEquipped == true {
			t.FailNow()
		}
	}
}

func TestFixupProfileFromProfileResponseMissingCharacterInventories(t *testing.T) {

	setup()
	response, err := getCurrentProfileResponse()
	if err != nil {
		t.FailNow()
	}
	response.Response.CharacterInventories = nil

	profile := fixupProfileFromProfileResponse(response)
	if profile == nil {
		t.FailNow()
	}
}

func TestLoadoutFromProfile(t *testing.T) {
	setup()

	setup()
	response, err := getCurrentProfileResponse()
	if err != nil {
		t.FailNow()
	}

	response.Response.CharacterInventories = nil
	response.Response.ProfileInventory = nil
	response.Response.ProfileCurrencies = nil
	response.Response.Profile = nil
	response.Response.ItemComponents = nil

	profile := fixupProfileFromProfileResponse(response)
	if profile == nil {
		t.FailNow()
	}

	loadout := loadoutFromProfile(profile)

	for equipmentBucket, item := range loadout {
		if item == nil {
			t.FailNow()
		}

		if _, ok := bucketHashLookup[equipmentBucket]; !ok {
			t.FailNow()
		}
	}
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
	f, err := os.Open("../test_data/bungie/" + name)
	if err != nil {
		return nil, err
	}

	return ioutil.ReadAll(f)
}
