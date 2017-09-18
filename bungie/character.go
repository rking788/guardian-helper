package bungie

import (
	"errors"
	"fmt"
	"time"
)

// Character will represent a single in-game character as well as platform membership data, emblems,
// last played date, and character class and race.
type Character struct {
	//https://bungie-net.github.io/multi/schema_Destiny-Entities-Characters-DestinyCharacterComponent.html#schema_Destiny-Entities-Characters-DestinyCharacterComponent
	MembershipID         string    `json:"membershipId"`
	MembershipType       int       `json:"membershipType"`
	CharacterID          string    `json:"characterId"`
	DateLastPlayed       time.Time `json:"dateLastPlayed"`
	EmblemBackgroundPath string    `json:"emblemBackgroundPath"`
	RaceHash             uint      `json:"raceHash"`
	ClassHash            uint      `json:"classHash"`
	ClassType            int       `json:"classType"`
	Light                int       `json:"light"`
}

func (c *Character) String() string {
	return fmt.Sprintf("Character{ID: %s, Race: %d, Class: %s, Power: %d}",
		c.MembershipID, c.RaceHash, classHashToName[c.ClassHash], c.Light)
}

// CharacterList represents a slice of Character pointers.
type CharacterList []*Character

// CharacterMap will be a map that contains Character values with characterID keys.
type CharacterMap map[string]*Character

// LastPlayedSort specifies a specific type for CharacterList that can be sorted by the date the character was last played.
type LastPlayedSort CharacterList

func (characters LastPlayedSort) Len() int { return len(characters) }
func (characters LastPlayedSort) Swap(i, j int) {
	characters[i], characters[j] = characters[j], characters[i]
}
func (characters LastPlayedSort) Less(i, j int) bool {
	return characters[i].DateLastPlayed.Before(characters[j].DateLastPlayed)
}

func (charList CharacterList) findCharacterFromID(characterID string) *Character {
	for _, char := range charList {
		if char.CharacterID == characterID {
			return char
		}
	}

	return nil
}

// findDestinationCharacter will find the first character matching the provided class name
// or an error if the account doesn't have a class of the specified type.
func (charList CharacterList) findDestinationCharacter(class string) (*Character, error) {

	if class == "vault" {
		return nil, nil
	}

	destinationHash := classNameToHash[class]
	for _, char := range charList {
		if char.ClassHash == destinationHash {
			return char, nil
		}
	}

	return nil, errors.New("could not find the specified destination character")
}
