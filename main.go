package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	"bitbucket.org/rking788/guardian-helper/bungie"
	"github.com/gin-gonic/gin"
	alexa "github.com/mikeflynn/go-alexa/skillserver"
)

func main() {

	port := os.Getenv("PORT")

	router := gin.New()
	router.Use(gin.Logger())

	router.GET("/", RootHandler)

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

	echoRequest := alexa.EchoRequest{}
	json.Unmarshal(requestBytes, &echoRequest)

	var response *alexa.EchoResponse
	if echoRequest.GetIntentName() == "CountItem" {
		item, _ := echoRequest.GetSlotValue("Item")
		lowerItem := strings.ToLower(item)
		response, err = bungie.CountItem(lowerItem, echoRequest.Session.User.AccessToken)
	} else if echoRequest.GetIntentName() == "TransferItem" {
		countStr, _ := echoRequest.GetSlotValue("Count")
		item, _ := echoRequest.GetSlotValue("Item")
		sourceClass, _ := echoRequest.GetSlotValue("Source")
		destinationClass, _ := echoRequest.GetSlotValue("Destination")
		output := fmt.Sprintf("Transferring %s of your %s from your %s to your %s", countStr, strings.ToLower(item), strings.ToLower(sourceClass), strings.ToLower(destinationClass))
		fmt.Println(output)
		response, err = bungie.TransferItem(strings.ToLower(item), echoRequest.Session.User.AccessToken, strings.ToLower(sourceClass), strings.ToLower(destinationClass), countStr)
	} else {
		response = alexa.NewEchoResponse()
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
