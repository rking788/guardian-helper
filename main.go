package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"

	"bitbucket.org/rking788/guardian-helper/alexa"
	"github.com/gin-gonic/gin"
	"github.com/mikeflynn/go-alexa/skillserver"
)

func main() {

	port := os.Getenv("PORT")

	router := gin.New()
	router.Use(gin.Logger())

	router.GET("/", RootHandler)
	router.StaticFile("/privacy.html", "./alexa/privacy.html")

	echo := router.Group("/echo")
	{
		echo.POST("/guardian-helper", EchoIntentHandler)
	}

	fmt.Println(fmt.Sprintf("Start listening on port(%s)", port))
	router.Run(":" + port)
}

// RootHandler is responsible for just responding with a String... that is all
// for now.
func RootHandler(ctx *gin.Context) {
	ctx.String(http.StatusOK, "Welcome!\n")
}

// Alexa skill related functions

// EchoIntentHandler is a handler method that is responsible for receiving the
// call from a Alexa command and returning the correct speech or cards.
func EchoIntentHandler(ctx *gin.Context) {

	requestBytes, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		fmt.Println("Error reading echo request!: ", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	echoRequest := &skillserver.EchoRequest{}
	json.Unmarshal(requestBytes, echoRequest)

	var response *skillserver.EchoResponse
	if echoRequest.GetIntentName() == "CountItem" {
		response = alexa.CountItem(echoRequest)
	} else if echoRequest.GetIntentName() == "TransferItem" {
		response = alexa.TransferItem(echoRequest)
	} else {
		response = skillserver.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, I did not understand your request.")
	}

	bytes, err := response.String()
	if err != nil {
		fmt.Println("Failed to convert EchoResponse to a byte slice.")
		return
	}

	ctx.String(http.StatusOK, string(bytes))
}

func dumpRequest(ctx *gin.Context) {

	data, err := httputil.DumpRequest(ctx.Request, true)
	if err != nil {
		fmt.Println("Failed to dump the request: ", err.Error())
		return
	}

	fmt.Println(string(data))
}
