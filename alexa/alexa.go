package alexa

// TODO: This file really needs a refactor. Endpoints that require a linked account
// should use some kind of middleware instead of having the check in individually handlers.

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/kpango/glg"
	"github.com/rking788/guardian-helper/bungie"
	"github.com/rking788/guardian-helper/db"
	"github.com/rking788/guardian-helper/trials"

	"strings"

	"github.com/garyburd/redigo/redis"
	"github.com/rking788/go-alexa/skillserver"
)

// Session is responsible for storing information related to a specific skill invocation.
// A session will remain open if the LaunchRequest was received.
type Session struct {
	ID                   string
	Action               string
	ItemName             string
	DestinationClassHash int
	SourceClassHash      int
	Quantity             int
}

var redisConnPool *redis.Pool

// InitEnv provides a package level initialization point for any work that is environment specific
func InitEnv(redisURL string) {
	redisConnPool = newRedisPool(redisURL)
}

// Redis related functions

func newRedisPool(addr string) *redis.Pool {
	// 25 is the maximum number of active connections for the Heroku Redis free tier
	return &redis.Pool{
		MaxIdle:     3,
		MaxActive:   25,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(addr) },
	}
}

// GetSession will attempt to read a session from the cache, if an existing one is not found, an empty session
// will be created with the specified sessionID.
func GetSession(sessionID string) (session *Session) {
	session = &Session{ID: sessionID}

	conn := redisConnPool.Get()
	defer conn.Close()

	key := fmt.Sprintf("sessions:%s", sessionID)
	reply, err := redis.String(conn.Do("GET", key))
	if err != nil {
		// NOTE: This is a normal situation, if the session is not stored in the cache, it will hit this condition.
		return
	}

	err = json.Unmarshal([]byte(reply), session)

	return
}

// SaveSession will persist the given session to the cache. This will allow support for long running
// Alexa sessions that continually prompt the user for more information.
func SaveSession(session *Session) {

	conn := redisConnPool.Get()
	defer conn.Close()

	sessionBytes, err := json.Marshal(session)
	if err != nil {
		glg.Errorf("Couldn't marshal session to string: %s", err.Error())
		return
	}

	key := fmt.Sprintf("sessions:%s", session.ID)
	_, err = conn.Do("SET", key, string(sessionBytes))
	if err != nil {
		glg.Errorf("Failed to set session: %s", err.Error())
	}
}

// ClearSession will remove the specified session from the local cache, this will be done
// when the user completes a full request session.
func ClearSession(sessionID string) {

	conn := redisConnPool.Get()
	defer conn.Close()

	key := fmt.Sprintf("sessions:%s", sessionID)
	_, err := conn.Do("DEL", key)
	if err != nil {
		glg.Errorf("Failed to delete the session from the Redis cache: %s", err.Error())
	}
}

// Handler is the type of function that should be used to respond to a specific intent.
type Handler func(*skillserver.EchoRequest) *skillserver.EchoResponse

// AuthWrapper is a handler function wrapper that will fail the chain of handlers if an access token was not provided
// as part of the Alexa request
func AuthWrapper(handler Handler) Handler {

	return func(req *skillserver.EchoRequest) *skillserver.EchoResponse {
		accessToken := req.Session.User.AccessToken
		if accessToken == "" {
			response := skillserver.NewEchoResponse()
			response.
				OutputSpeech("Sorry Guardian, it looks like your Bungie.net account needs to be linked in the Alexa app.").
				LinkAccountCard()
			return response
		}

		return handler(req)
	}
}

// WelcomePrompt is responsible for prompting the user with information about what they can ask
// the skill to do.
func WelcomePrompt(echoRequest *skillserver.EchoRequest) (response *skillserver.EchoResponse) {
	response = skillserver.NewEchoResponse()

	response.OutputSpeech("Welcome Guardian, would you like to equip max light, unload engrams, or transfer an item to a specific character, " +
		"find out how many of an item you have, or ask about Trials of Osiris?").
		Reprompt("Do you want to equip max light, unload engrams, transfer an item, find out how much of an item you have, or ask about Trials of Osiris?").
		EndSession(false)

	return
}

// HelpPrompt provides the required information to satisfy the HelpIntent built-in Alexa intent. This should
// provider information to the user to let them know what the skill can do without providing exact commands.
func HelpPrompt(echoRequest *skillserver.EchoRequest) (response *skillserver.EchoResponse) {
	response = skillserver.NewEchoResponse()

	response.OutputSpeech("Welcome Guardian, I am here to help manage your Destiny in-game inventory. You can ask " +
		"me to equip your max light loadout, unload engrams from your inventory, or transfer items between your available " +
		"characters including the vault. You can also ask how many of an " +
		"item you have. Trials of Osiris statistics provided by Trials Report are available too.").
		EndSession(false)

	return
}

// CountItem calls the Bungie API to see count the number of Items on all characters and
// in the vault.
func CountItem(echoRequest *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	accessToken := echoRequest.Session.User.AccessToken
	item, _ := echoRequest.GetSlotValue("Item")
	lowerItem := strings.ToLower(item)
	response, err := bungie.CountItem(lowerItem, accessToken)
	if err != nil {
		glg.Errorf("Error counting the number of items: %s", err.Error())
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, an error occurred counting that item.")
	}

	return
}

// TransferItem will attempt to transfer either a specific quantity or all of a
// specific item to a specified character. The item name and destination are the
// required fields. The quantity and source are optional.
func TransferItem(request *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	accessToken := request.Session.User.AccessToken
	countStr, _ := request.GetSlotValue("Count")
	count := -1
	if countStr != "" {
		tempCount, ok := strconv.Atoi(countStr)
		if ok != nil {
			response = skillserver.NewEchoResponse()
			response.OutputSpeech("Sorry Guardian, I didn't understand the number you asked to be transferred. Do not specify a quantity if you want all to be transferred.")
			return
		}

		if tempCount <= 0 {
			output := fmt.Sprintf("Sorry Guardian, you need to specify a positive, non-zero number to be transferred, not %d", tempCount)
			response.OutputSpeech(output)
			return
		}

		count = tempCount
	}

	item, _ := request.GetSlotValue("Item")
	sourceClass, _ := request.GetSlotValue("Source")
	destinationClass, _ := request.GetSlotValue("Destination")
	if destinationClass == "" {
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, you must specify a destination for the items to be transferred.")
		return
	}

	glg.Infof("Transferring %d of your %s from your %s to your %s", count, strings.ToLower(item), strings.ToLower(sourceClass), strings.ToLower(destinationClass))
	response, err := bungie.TransferItem(strings.ToLower(item), accessToken, strings.ToLower(sourceClass), strings.ToLower(destinationClass), count)
	if err != nil {
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, an error occurred trying to transfer that item.")
		return
	}

	return
}

// MaxPower will equip the loadout on the current character that provides the maximum amount of power.
func MaxPower(request *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	accessToken := request.Session.User.AccessToken
	response, err := bungie.EquipMaxLightGear(accessToken)
	if err != nil {
		glg.Errorf("Error occurred equipping max light: %s", err.Error())
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, an error occurred equipping your max light gear.")
	}

	return
}

// UnloadEngrams will take all engrams on all of the current user's characters and transfer them all to the
// vault to allow the player to continue farming.
func UnloadEngrams(request *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	accessToken := request.Session.User.AccessToken
	response, err := bungie.UnloadEngrams(accessToken)
	if err != nil {
		glg.Errorf("Error occurred unloading engrams: %s", err.Error())
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, an error occurred moving your engrams.")
	}

	return
}

// DestinyJoke will return the desired text for a random joke from the database.
func DestinyJoke(request *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	response = skillserver.NewEchoResponse()
	setup, punchline, err := db.GetRandomJoke()
	if err != nil {
		glg.Errorf("Error loading joke from DB: %s", err.Error())
		response.OutputSpeech("Sorry Guardian, I was unable to load a joke right now.")
		return
	}

	builder := skillserver.NewSSMLTextBuilder()
	builder.AppendPlainSpeech(setup).
		AppendBreak("2s", "medium", "").
		AppendPlainSpeech(punchline)
	response.OutputSpeechSSML(builder.Build())

	return
}

// CreateLoadout will determine the user's current character and create a new loadout based on
// their currently equipped items. This loadout is then serialized and persisted to a
// persistent store.
func CreateLoadout(request *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	response = skillserver.NewEchoResponse()

	glg.Debugf("Found dialog state = %s", request.GetDialogState())
	if request.GetDialogState() != skillserver.DialogCompleted {
		// The user still needs to provide a name for the new loadout to be created
		response.DialogDelegate(nil)
		return
	}

	glg.Debugf("Found intent confirmation status = %s", request.GetIntentConfirmationStatus())
	intentConfirmation := request.GetIntentConfirmationStatus()
	if intentConfirmation == skillserver.ConfirmationDenied {
		// The user does NOT want to overwrite the existing loadout with the same name
		return
	}

	accessToken := request.Session.User.AccessToken
	loadoutName, _ := request.GetSlotValue("Name")
	if loadoutName == "" {
		response.OutputSpeech("Sorry Guardian, you must specify a name for the loadout being saved.")
	}

	var err error
	response, err = bungie.CreateLoadoutForCurrentCharacter(accessToken, loadoutName,
		intentConfirmation == "CONFIRMED")

	if err != nil {
		glg.Errorf("Error occurred creating loadout: %s", err.Error())
		response.OutputSpeech("Sorry Guardian, an error occurred saving your loadout.")
	}

	return
}

func EquipNamedLoadout(request *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	accessToken := request.Session.User.AccessToken
	loadoutName, _ := request.GetSlotValue("Name")
	if loadoutName == "" {
		response.OutputSpeech("Sorry Guardian, you must specify a name for the loadout being equipped.")
	}

	response, err := bungie.EquipNamedLoadout(accessToken, loadoutName)

	if err != nil {
		glg.Errorf("Error occurred creating loadout: %s", err.Error())
		response.OutputSpeech("Sorry Guardian, an error occurred equipping your loadout.")
	}

	return
}

/*
 * Trials of Osiris data
 */

// CurrentTrialsMap will return a brief description of the current map in the active Trials of Osiris week.
func CurrentTrialsMap(request *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	response, err := trials.GetCurrentMap()
	if err != nil {
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, I cannot access this information right now, please try again later.")
		return
	}

	return
}

// CurrentTrialsWeek will return a brief description of the current map in the active Trials of Osiris week.
func CurrentTrialsWeek(request *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	accessToken := request.Session.User.AccessToken
	response, err := trials.GetCurrentWeek(accessToken)
	if err != nil {
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, I cannot access this information right now, please try again later.")
		return
	}

	return
}

// PopularWeapons will check Trials Report for the most popular specific weapons for the current week.
func PopularWeapons(request *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	response, err := trials.GetWeaponUsagePercentages()
	if err != nil {
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, I cannot access this information at this time, please try again later")
		return
	}

	return
}

// PersonalTopWeapons will check Trials Report for the most used weapons for the current user.
func PersonalTopWeapons(request *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	accessToken := request.Session.User.AccessToken
	response, err := trials.GetPersonalTopWeapons(accessToken)
	if err != nil {
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, I cannot access this information at this time, please try again later")
		return
	}

	return
}

// PopularWeaponTypes will return info about what classes of weapons are getting
// the most kills in Trials of Osiris.
func PopularWeaponTypes(echoRequest *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	response, err := trials.GetPopularWeaponTypes()
	if err != nil {
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, I cannot access this information at this time, pleast try again later")
		return
	}

	return
}
