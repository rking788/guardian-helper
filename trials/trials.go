package trials

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	"time"

	"strconv"

	"github.com/mikeflynn/go-alexa/skillserver"
	"github.com/rking788/guardian-helper/bungie"
)

type CurrentMap struct {
	Name       string `json:"activityName"`
	WeekNumber string `json:"week"`
	StartDate  string `json:"start_date"`
}

type CurrentWeek struct {
	Matches string `json:"matches"`
	Losses  string `json:"losses"`
	KD      string `json:"kd"`
}

// GetCurrentMap will make a request to the Trials Report API endpoint and
// return an Alexa response describing the current map.
func GetCurrentMap() (*skillserver.EchoResponse, error) {

	response := skillserver.NewEchoResponse()

	req, _ := http.NewRequest("GET", TrialsCurrentMapEndpoint, nil)
	req.Header.Add("Content-Type", "application/json")

	mapResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Failed to read the current map response from Trials Report!: ", err.Error())
		return nil, err
	}
	defer mapResponse.Body.Close()

	currentMaps := make([]CurrentMap, 0, 1)
	err = json.NewDecoder(mapResponse.Body).Decode(&currentMaps)
	if err != nil {
		fmt.Println("Error parsing trials report response: ", err.Error())
		return nil, err
	}
	fmt.Printf("Current map response from Trials Report: %+v\n", currentMaps[0])
	start, err := time.Parse("2006-01-02 15:04:05", currentMaps[0].StartDate)

	response.OutputSpeech(fmt.Sprintf("According to Trials Report, the current Trials of Osiris map beginning %s %d is %s, goodluck Guardian.", start.Month().String(), start.Day(), currentMaps[0].Name))

	return response, nil
}

// GetCurrentWeek is responsible for requesting the players stats from the current week from Trials Report.
func GetCurrentWeek(token string) (*skillserver.EchoResponse, error) {
	response := skillserver.NewEchoResponse()

	membershipID, err := findMembershipID(token)
	fmt.Printf("Found membershipID: %s\n", membershipID)

	url := fmt.Sprintf(TrialsCurrentWeekEndpointFmt, membershipID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Content-Type", "application/json")

	mapResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Failed to read the current week stats response from Trials Report!: ", err.Error())
		return nil, err
	}
	defer mapResponse.Body.Close()

	currentWeeks := make([]CurrentWeek, 0, 1)
	err = json.NewDecoder(mapResponse.Body).Decode(&currentWeeks)
	if err != nil {
		fmt.Println("Error parsing trials report response: ", err.Error())
		return nil, err
	}
	fmt.Printf("Current week response from Trials Report: %+v\n", currentWeeks[0])

	matches, _ := strconv.ParseInt(currentWeeks[0].Matches, 10, 32)
	if matches != 0 {
		losses, _ := strconv.ParseInt(currentWeeks[0].Losses, 10, 32)
		wins := matches - losses
		kd := currentWeeks[0].KD
		response.OutputSpeech(fmt.Sprintf("So far you have played %d matches with %d wins, %d losses and a combined KD of %s", matches, wins, losses, kd))
	} else {
		response.OutputSpeech("You have not yet played any Trials of Osiris matches this week guardian.")
	}

	return response, nil
}

// findMembershipID is a helper function for loading the membership ID from the currently
// linked account, this eventually should take platform into account.
func findMembershipID(token string) (string, error) {

	client := bungie.NewClient(token, os.Getenv("BUNGIE_API_KEY"))
	currentAccount, err := client.GetCurrentAccount()
	if err != nil {
		fmt.Println("Error loading current account info from Bungie.net: ", err.Error())
		return "", err
	} else if currentAccount.Response == nil || currentAccount.Response.DestinyAccounts == nil ||
		len(currentAccount.Response.DestinyAccounts) == 0 {
		return "", errors.New("No linked Destiny account found on Bungie.net")
	}

	// TODO: This should take the platform into account instead of just defaulting to the first one.
	return currentAccount.Response.DestinyAccounts[0].UserInfo.MembershipID, nil
}
