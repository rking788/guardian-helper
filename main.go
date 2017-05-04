package main

import (
	"fmt"
	"net/http/httputil"
	"os"

	"bitbucket.org/rking788/guardian-helper/alexa"

	"github.com/gin-gonic/gin"
	"github.com/rking788/go-alexa/skillserver"
)

// Applications is a definition of the Alexa applications running on this server.
var Applications = map[string]interface{}{
	"/echo/guardian-helper": skillserver.EchoApplication{ // Route
		AppID:    os.Getenv("ALEXA_APP_ID"), // Echo App ID from Amazon Dashboard
		OnIntent: EchoIntentHandler,
		OnLaunch: EchoIntentHandler,
	},
}

func main() {

	port := os.Getenv("PORT")

	fmt.Println(fmt.Sprintf("Start listening on port(%s)", port))
	skillserver.Run(Applications, port)
}

// Alexa skill related functions

// EchoIntentHandler is a handler method that is responsible for receiving the
// call from a Alexa command and returning the correct speech or cards.
func EchoIntentHandler(echoRequest *skillserver.EchoRequest, echoResponse *skillserver.EchoResponse) {

	var response *skillserver.EchoResponse

	if echoRequest.GetIntentName() == "CountItem" {
		response = alexa.CountItem(echoRequest)
	} else if echoRequest.GetIntentName() == "TransferItem" {
		response = alexa.TransferItem(echoRequest)
	} else {
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, I did not understand your request.")
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
