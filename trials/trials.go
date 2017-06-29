package trials

import (
	"encoding/json"
	"fmt"
	"net/http"

	"time"

	"github.com/mikeflynn/go-alexa/skillserver"
)

type CurrentMap struct {
	Name       string `json:"activityName"`
	WeekNumber string `json:"week"`
	StartDate  string `json:"start_date"`
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
