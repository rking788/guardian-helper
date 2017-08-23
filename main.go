package main

import (
	"fmt"
	"net/http/httputil"
	"os"

	"github.com/rking788/guardian-helper/bungie"

	"github.com/rking788/guardian-helper/alexa"

	"github.com/gin-gonic/gin"
	"github.com/mikeflynn/go-alexa/skillserver"
)

// Applications is a definition of the Alexa applications running on this server.
var (
	Applications = map[string]interface{}{
		"/echo/guardian-helper": skillserver.EchoApplication{ // Route
			AppID:          os.Getenv("ALEXA_APP_ID"), // Echo App ID from Amazon Dashboard
			OnIntent:       EchoIntentHandler,
			OnLaunch:       EchoIntentHandler,
			OnSessionEnded: EchoSessionEndedHandler,
		},
	}
)

func main() {

	port := os.Getenv("PORT")

	err := bungie.PopulateEngramHashes()
	if err != nil {
		return
	}

	fmt.Println(fmt.Sprintf("Start listening on port(%s)", port))
	skillserver.Run(Applications, port)
}

// Alexa skill related functions

// EchoSessionEndedHandler is responsible for cleaning up an open session since the user has quit the session.
func EchoSessionEndedHandler(echoRequest *skillserver.EchoRequest, echoResponse *skillserver.EchoResponse) {
	*echoResponse = *skillserver.NewEchoResponse()

	alexa.ClearSession(echoRequest.GetSessionID())
}

// EchoIntentHandler is a handler method that is responsible for receiving the
// call from a Alexa command and returning the correct speech or cards.
func EchoIntentHandler(echoRequest *skillserver.EchoRequest, echoResponse *skillserver.EchoResponse) {

	var response *skillserver.EchoResponse

	// See if there is an existing session, or create a new one.
	session := alexa.GetSession(echoRequest.GetSessionID())
	alexa.SaveSession(session)

	intentName := echoRequest.GetIntentName()

	fmt.Printf("Launching with RequestType: %s, IntentName: %s\n", echoRequest.GetRequestType(), intentName)

	if echoRequest.GetRequestType() == "LaunchRequest" {
		response = alexa.WelcomePrompt(echoRequest)
	} else if intentName == "CountItem" {
		response = alexa.CountItem(echoRequest)
	} else if intentName == "TransferItem" {
		response = alexa.TransferItem(echoRequest)
	} else if intentName == "TrialsCurrentMap" {
		response = alexa.CurrentTrialsMap(echoRequest)
	} else if intentName == "TrialsCurrentWeek" {
		response = alexa.CurrentTrialsWeek(echoRequest)
	} else if intentName == "TrialsTopWeapons" {
		response = alexa.PopularWeapons(echoRequest)
	} else if intentName == "TrialsPersonalTopWeapons" {
		response = alexa.PersonalTopWeapons(echoRequest)
	} else if intentName == "TrialsPopularWeaponTypes" {
		response = alexa.PopularWeaponTypes()
	} else if intentName == "UnloadEngrams" {
		response = alexa.UnloadEngrams(echoRequest)
	} else if intentName == "AMAZON.HelpIntent" {
		response = alexa.HelpPrompt(echoRequest)
	} else if intentName == "AMAZON.StopIntent" {
		response = skillserver.NewEchoResponse()
	} else if intentName == "AMAZON.CancelIntent" {
		response = skillserver.NewEchoResponse()
	} else {
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, I did not understand your request.")
	}

	if response.Response.ShouldEndSession {
		alexa.ClearSession(session.ID)
	}

	*echoResponse = *response
}

func dumpRequest(ctx *gin.Context) {

	data, err := httputil.DumpRequest(ctx.Request, true)
	if err != nil {
		fmt.Println("Failed to dump the request: ", err.Error())
		return
	}

	fmt.Println(string(data))
}
