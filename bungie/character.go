package bungie

import (
	"errors"
	"time"
)

// Character will represent a single character entry returned by the /Items endpoint
type Character struct {
	CharacterBase *CharacterBase
	// NOTE: The rest is probably unused at least for the transferring items command
}

// CharacterBase represents the base data for a character entry
// returned by the /Items endpoint.
type CharacterBase struct {
	MembershipID           string    `json:"membershipId"`
	MembershipType         uint      `json:"membershipType"`
	CharacterID            string    `json:"characterId"`
	DateLastPlayed         time.Time `json:"dateLastPlayed"`
	PowerLevel             uint      `json:"powerLevel"`
	RaceHash               uint      `json:"raceHash"`
	GenderHash             uint      `json:"genderHash"`
	ClassHash              uint      `json:"classHash"`
	CurrentActivityHash    uint      `json:"currentActivityHash"`
	LastCompletedStoryHash uint      `json:"lastCompletedStoryHash"`
	GenderType             uint      `json:"genderType"`
	ClassType              uint      `json:"ClassType"`
}

func findDestinationCharacter(characters []*Character, class string) (*Character, error) {

	if class == "vault" {
		return nil, nil
	}

	destinationHash := classNameToHash[class]
	for _, char := range characters {
		if char.CharacterBase.ClassHash == destinationHash {
			return char, nil
		}
	}

	return nil, errors.New("could not find the specified destination character")
}
