package alexa

import (
	"fmt"
	"strconv"

	"bitbucket.org/rking788/guardian-helper/bungie"

	"strings"

	"github.com/mikeflynn/go-alexa/skillserver"
)

// CountItem calls the Bungie API to see count the number of Items on all characters and
// in the vault.
func CountItem(echoRequest *skillserver.EchoRequest, echoResponse *skillserver.EchoResponse) {

	accessToken := echoRequest.Session.User.AccessToken
	if accessToken == "" {
		echoResponse.
			OutputSpeech("Sorry Guardian, it looks like your Bungie.net account needs to be linked in the Alexa app.").
			LinkAccountCard()
		return
	}

	item, _ := echoRequest.GetSlotValue("Item")
	lowerItem := strings.ToLower(item)
	response, err := bungie.CountItem(lowerItem, accessToken)
	if err != nil {
		fmt.Println("Error counting the number of items: ", err.Error())
		echoResponse.OutputSpeech("Sorry Guardian, an error occurred counting that item.")
		return
	}

	*echoResponse = *response
}

// TransferItem will attempt to transfer either a specific quantity or all of a
// specific item to a specified character. The item name and destination are the
// required fields. The quantity and source are optional.
func TransferItem(request *skillserver.EchoRequest, echoResponse *skillserver.EchoResponse) {

	accessToken := request.Session.User.AccessToken
	if accessToken == "" {
		echoResponse.
			OutputSpeech("Sorry Guardian, it looks like your Bungie.net account needs to be linked in the Alexa app.").
			LinkAccountCard()
		return
	}

	countStr, _ := request.GetSlotValue("Count")
	count := -1
	if countStr != "" {
		if tempCount, ok := strconv.Atoi(countStr); ok != nil {
			if tempCount <= 0 {
				output := fmt.Sprintf("Sorry Guardian, you need to specify a positive, non-zero count to be transferred, not %d", tempCount)
				fmt.Println(output)
				echoResponse.OutputSpeech(output)
				return
			}

			count = tempCount
		} else {
			echoResponse.OutputSpeech("Sorry Guardian, I didn't understand the number you asked to be transferred. If you don't specify a quantity then all will be transferred.")
			return
		}
	}

	item, _ := request.GetSlotValue("Item")
	sourceClass, _ := request.GetSlotValue("Source")
	destinationClass, _ := request.GetSlotValue("Destination")
	output := fmt.Sprintf("Transferring %d of your %s from your %s to your %s", count, strings.ToLower(item), strings.ToLower(sourceClass), strings.ToLower(destinationClass))
	fmt.Println(output)
	response, err := bungie.TransferItem(strings.ToLower(item), accessToken, strings.ToLower(sourceClass), strings.ToLower(destinationClass), count)
	if err != nil {
		echoResponse.OutputSpeech("Sorry Guardian, an error occurred trying to transfer that item.")
		return
	}

	*echoResponse = *response
}
