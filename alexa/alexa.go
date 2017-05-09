package alexa

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"bitbucket.org/rking788/guardian-helper/bungie"

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

var (
	redisConnPool = newRedisPool(os.Getenv("REDIS_URL"))
)

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
		fmt.Println("Failed to read session from cache:", err.Error())
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
		fmt.Println("Couldn't marshal session to string: ", err.Error())
		return
	}

	key := fmt.Sprintf("sessions:%s", session.ID)
	_, err = conn.Do("SET", key, string(sessionBytes))
	if err != nil {
		fmt.Println("Failed to set session: ", err.Error())
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
		fmt.Println("Failed to delete the session from the Redis cache: ", err.Error())
	}
}

// WelcomePrompt is responsible for prompting the user with information about what they can ask
// the skill to do.
func WelcomePrompt(echoRequest *skillserver.EchoRequest) (response *skillserver.EchoResponse) {
	response = skillserver.NewEchoResponse()

	response.OutputSpeech("Welcome Guardian, would you like to transfer an item to a specific character or find out how many of an item you have?").
		Reprompt("Do you want to transfer an item or find out how much of an item you have?").
		EndSession(false)

	return
}

// CountItem calls the Bungie API to see count the number of Items on all characters and
// in the vault.
func CountItem(echoRequest *skillserver.EchoRequest) (response *skillserver.EchoResponse) {

	accessToken := echoRequest.Session.User.AccessToken
	if accessToken == "" {
		response = skillserver.NewEchoResponse()
		response.
			OutputSpeech("Sorry Guardian, it looks like your Bungie.net account needs to be linked in the Alexa app.").
			LinkAccountCard()
		return
	}

	item, _ := echoRequest.GetSlotValue("Item")
	lowerItem := strings.ToLower(item)
	response, err := bungie.CountItem(lowerItem, accessToken)
	if err != nil {
		fmt.Println("Error counting the number of items: ", err.Error())
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
	if accessToken == "" {
		response = skillserver.NewEchoResponse()
		response.
			OutputSpeech("Sorry Guardian, it looks like your Bungie.net account needs to be linked in the Alexa app.").
			LinkAccountCard()
		return
	}

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
			fmt.Println(output)
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
	output := fmt.Sprintf("Transferring %d of your %s from your %s to your %s", count, strings.ToLower(item), strings.ToLower(sourceClass), strings.ToLower(destinationClass))
	fmt.Println(output)
	response, err := bungie.TransferItem(strings.ToLower(item), accessToken, strings.ToLower(sourceClass), strings.ToLower(destinationClass), count)
	if err != nil {
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, an error occurred trying to transfer that item.")
		return
	}

	return
}
