package main

import (
	alexa "github.com/mikeflynn/go-alexa/skillserver"
)

// CountItem calls the Bungie API to see count the number of Items on all characters and
// in the vault.
func CountItem(itemName, accessToken, apiKey string) alexa.EchoResponse {

	return alexa.EchoResponse{}
}

func GetPrivacyPolicyHTML() string {
	return `
		<h1></h1>
		<p>
		</p>
	`
}
