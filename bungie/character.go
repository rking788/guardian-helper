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

// CharacterList represents a slice of Character pointers.
type CharacterList []*Character

// LastPlayedSort specifies a specific type for CharacterList that can be sorted by the date the character was last played.
type LastPlayedSort CharacterList

func (characters LastPlayedSort) Len() int { return len(characters) }
func (characters LastPlayedSort) Swap(i, j int) {
	characters[i], characters[j] = characters[j], characters[i]
}
func (characters LastPlayedSort) Less(i, j int) bool {
	return characters[i].CharacterBase.DateLastPlayed.Before(characters[j].CharacterBase.DateLastPlayed)
}

// findDestinationCharacter will find the first character matching the provided class name
// or an error if the account doesn't have a class of the specified type.
func findDestinationCharacter(characters CharacterList, class string) (*Character, error) {

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

// finDestinationCharacterIndex will find the first index of a character with the provided class name
// or an error if the account doesn't have a character of the specified class.
func findDestinationCharacterIndex(characters CharacterList, class string) (int, error) {

	if class == "vault" {
		return -1, nil
	}

	destinationHash := classNameToHash[class]
	for index, char := range characters {
		if char.CharacterBase.ClassHash == destinationHash {
			return index, nil
		}
	}

	return -2, errors.New("No character of that type on this account")
}
