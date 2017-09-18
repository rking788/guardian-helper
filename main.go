package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/kpango/glg"
	"github.com/rking788/guardian-helper/alexa"

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
		"/health": skillserver.StdApplication{
			Methods: "GET",
			Handler: healthHandler,
		},
	}
)

// AlexaHandlers are the handler functions mapped by the intent name that they should handle.
var (
	AlexaHandlers = map[string]alexa.Handler{
		"CountItem":                alexa.AuthWrapper(alexa.CountItem),
		"TransferItem":             alexa.AuthWrapper(alexa.TransferItem),
		"TrialsCurrentMap":         alexa.CurrentTrialsMap,
		"TrialsCurrentWeek":        alexa.AuthWrapper(alexa.CurrentTrialsWeek),
		"TrialsTopWeapons":         alexa.PopularWeapons,
		"TrialsPopularWeaponTypes": alexa.PopularWeaponTypes,
		"TrialsPersonalTopWeapons": alexa.AuthWrapper(alexa.PersonalTopWeapons),
		"UnloadEngrams":            alexa.AuthWrapper(alexa.UnloadEngrams),
		"EquipMaxLight":            alexa.AuthWrapper(alexa.MaxLight),
		"DestinyJoke":              alexa.DestinyJoke,
		"AMAZON.HelpIntent":        alexa.HelpPrompt,
	}
)

var memprofile = flag.String("memprofile", "", "write memory profile to this file")

func init() {
	ConfigureLogging()
}

func main() {

	flag.Parse()
	port := os.Getenv("PORT")

	defer CloseLogger()

	//bungie.EquipMaxLightGear("access-token")

	// c := make(chan os.Signal, 1)
	// signal.Notify(c, os.Interrupt)
	// go func() {
	// 	for _ = range c {
	// 		if *memprofile != "" {
	// 			f, err := os.Create(*memprofile)
	// 			if err != nil {
	// 				log.Fatal(err)
	// 			}
	// 			pprof.WriteHeapProfile(f)
	// 			f.Close()
	// 			os.Exit(1)
	// 			return
	// 		}
	// 	}
	// }()

	err := skillserver.RunSSL(Applications, port, "/etc/letsencrypt/live/warmind.network/fullchain.pem", "/etc/letsencrypt/live/warmind.network/privkey.pem")
	if err != nil {
		glg.Errorf("Error starting the application!: %s", err.Error())
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Up"))
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

	// Time the intent handler to determine if it is taking longer than normal
	startTime := time.Now()
	defer func(start time.Time) {
		glg.Infof("IntentHandler execution time: %v", time.Since(start))
	}(startTime)

	var response *skillserver.EchoResponse

	// See if there is an existing session, or create a new one.
	session := alexa.GetSession(echoRequest.GetSessionID())
	alexa.SaveSession(session)

	intentName := echoRequest.GetIntentName()

	glg.Infof("RequestType: %s, IntentName: %s", echoRequest.GetRequestType(), intentName)

	handler, ok := AlexaHandlers[intentName]
	if echoRequest.GetRequestType() == "LaunchRequest" {
		response = alexa.WelcomePrompt(echoRequest)
	} else if intentName == "AMAZON.StopIntent" {
		response = skillserver.NewEchoResponse()
	} else if intentName == "AMAZON.CancelIntent" {
		response = skillserver.NewEchoResponse()
	} else if ok {
		response = handler(echoRequest)
	} else {
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, I did not understand your request.")
	}

	if response.Response.ShouldEndSession {
		alexa.ClearSession(session.ID)
	}

	*echoResponse = *response
}

// func dumpRequest(ctx *gin.Context) {

// 	data, err := httputil.DumpRequest(ctx.Request, true)
// 	if err != nil {
// 		glg.Errorf("Failed to dump the request: %s", err.Error())
// 		return
// 	}

// 	glg.Debug(string(data))
// }
