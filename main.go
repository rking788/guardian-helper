package main

import (
	"encoding/base64"
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

// BungieAPIResponse is a data type that represents the JSON response
// from a Bungie API call.
type BungieAPIResponse struct {
	Response        *Response   `json:"Response"`
	ErrorCode       int         `json:"ErrorCode"`
	ThrottleSeconds int         `json:"ThrottleSeconds"`
	ErrorStatus     string      `json:"ErrorStatus"`
	Message         string      `json:"Message"`
	MessageData     interface{} `json:"MessageData"`
}

// Response represents the response portion of a Bungie API response.
type Response struct {
	AccessToken  *Token `json:"accessToken"`
	RefreshToken *Token `json:"refreshToken"`
	Scope        int    `json:"scope"`
}

// Token is a data type representing the fields of an access token,
// could be the access_token itself or a refresh token.
type Token struct {
	Value   string `json:"Value"`
	ReadyIn int    `json:"readyin"`
	Expires int    `json:"expires"`
}

var redirectURI string
var code string
var state string

// RootHandler is responsible for just responding with a String... that is all
// for now.
func RootHandler(ctx *gin.Context) {
	ctx.String(http.StatusOK, "Welcome!\n")
}

func main() {

	port := os.Getenv("PORT")

	router := gin.New()
	router.Use(gin.Logger())

	router.GET("/", RootHandler)

	// From Alexa to be redirected to the Bungie Authorization URL
	router.GET("/auth", AuthGetHandler)

	router.GET("auth-code-response", AuthCodeResponseHandler)

	router.POST("tokens", TokenHandler)

	//app := alexa.EchoApplication{AppID: "", Handler: EchoIntentHandler}
	echo := router.Group("/echo")
	{
		echo.POST("/guardian-helper", EchoIntentHandler)
	}

	fmt.Println(fmt.Sprintf("Start listening on port(%s)", port))

	router.Run(":" + port)
}

// AuthGetHandler is a handler function that is responsible for redirecting
// the Alexa app to the Bungie authorization page for the app during
// account linking.
func AuthGetHandler(ctx *gin.Context) {
	dumpRequest(ctx)

	redirectURI = ctx.Query("redirect_uri")
	state = ctx.Query("state")

	ctx.Redirect(http.StatusFound, bungie.AppAuthURL)
}

// AuthCodeResponseHandler is a handler that will receive a callback from the
// Bungie authorization page with the Authorization code to be sent back to the
// Alexa redirect URI specified in a previous request.
func AuthCodeResponseHandler(ctx *gin.Context) {
	dumpRequest(ctx)
	uri := fmt.Sprintf("%s?state=%s&code=%s", redirectURI, state, ctx.Query("code"))

	fmt.Println("Sending redirection to: ", uri)
	ctx.Redirect(http.StatusFound, uri)
}

// TokenHandler is a handler responsible for receiving a call from the Alexa Platform
// and requesting a new access token from the Bungie API. This will generate
// a new access token and can also be used to refresh a previous token.
func TokenHandler(ctx *gin.Context) {
	dumpRequest(ctx)

	ctx.Request.ParseForm()

	if ctx.Request.Form.Get("refresh_token") != "" {
		fmt.Println("Getting tokens from refresh token...")
		getAccessTokenWithRefreshToken(ctx, ctx.Request.Form.Get("refresh_token"))
	} else {
		fmt.Println("Getting tokens from auth code...")
		getAccessTokenWithAuthCode(ctx, ctx.Request.Form.Get("code"))
	}
}

func getAccessTokenWithRefreshToken(ctx *gin.Context, refreshToken string) {

	apiKey := readAPIKey(ctx)

	bodyJSON := make(map[string]string)
	bodyJSON["refreshToken"] = refreshToken
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		fmt.Println("Failed to marshal JSON: ", err.Error())
		return
	}

	client := http.Client{}
	req, err := http.NewRequest("POST", bungie.TokensFromRefreshTokenURL, strings.NewReader(string(body)))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-API-Key", apiKey)

	tokenResponse, err := client.Do(req)
	tokenBytes, err := ioutil.ReadAll(tokenResponse.Body)
	if err != nil {
		fmt.Println("Failed to read the token response from Bungie!: ", err.Error())
		return
	}

	tokenJSON := BungieAPIResponse{}
	json.Unmarshal(tokenBytes, &tokenJSON)

	fmt.Printf("Unmarshal-ed response from Bungie API: %+v\n", tokenJSON)

	if tokenJSON.ErrorStatus != "Success" || tokenJSON.Message != "Ok" {
		fmt.Println("Got an invalid response back")
		return
	}

	response := make(map[string]interface{})
	response["access_token"] = tokenJSON.Response.AccessToken.Value
	response["expires_in"] = tokenJSON.Response.AccessToken.Expires
	response["refresh_token"] = tokenJSON.Response.RefreshToken.Value

	fmt.Printf("Sending response to Alexa refreshing tokens: %+v\n", response)

	ctx.JSON(http.StatusOK, response)
}

func getAccessTokenWithAuthCode(ctx *gin.Context, code string) {

	apiKey := readAPIKey(ctx)

	bodyJSON := make(map[string]string)
	bodyJSON["code"] = code
	body, err := json.Marshal(bodyJSON)
	if err != nil {
		fmt.Println("Failed to marshal JSON: ", err.Error())
		return
	}

	client := http.Client{}
	req, err := http.NewRequest("POST", bungie.TokensFromAuthCodeURL, strings.NewReader(string(body)))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-API-Key", apiKey)

	tokenResponse, err := client.Do(req)
	tokenBytes, err := ioutil.ReadAll(tokenResponse.Body)
	if err != nil {
		fmt.Println("Failed to read the token response from Bungie!: ", err.Error())
		return
	}

	tokenJSON := BungieAPIResponse{}
	json.Unmarshal(tokenBytes, &tokenJSON)

	fmt.Printf("Unmarshal-ed response from Bungie API: %+v\n", tokenJSON)

	if tokenJSON.ErrorStatus != "Success" || tokenJSON.Message != "Ok" {
		fmt.Println("Got an invalid response back")
		return
	}

	response := make(map[string]interface{})
	response["access_token"] = tokenJSON.Response.AccessToken.Value
	response["expires_in"] = tokenJSON.Response.AccessToken.Expires
	response["refresh_token"] = tokenJSON.Response.RefreshToken.Value
	//response["access_token"] = "CN8LEoYCACC8pZp6gwaYHYD8W/53hYuiHiHNP7UDM9TaX3NHzGP1tOAAAACVbQXuNzAxfigs0v9NtD75pNptRHCZNyE0f58AbwSxYF5RolwwZGK/om2v8hzeYQSu2Rq2SakapPYRCVj+GaTE48TxpQSbDr8RzRQuGujkOoBAk3FjRoT7teIg6HX0N2nFIcn7mC+99OpNMKy+8GU/VIR5f/GTpB+jn8l7lPonWBHo9JZaoO84LpAntl172Yqrxw61QGLVoZClFfLBRJFI8ZpGaXDwXM3nc1x2yLiUHMgzJ5zHdXP6x7jg5Nq1DKR2ONJ7TRF4UZPZBl8wCJFlurj47ts2UOYdSRCnw/v57Q=="
	//response["refresh_token"] = "CN8LEoYCACCXqZL3oDZB1NHal9P969bLKTMydr2Q/GKPA/C1dMm5YOAAAAC9SGaI/asp9lnYmEPUApiRjHznkWzJd48Wlpqaic+9HfedgpUW14yC/ltKea3Sx/SyP9pTZovYUJwihrC+NaYAMVllfzNyHG+CyCjVeIcnQZg6qOEI3PC+OK3QhQvXt1H1BDCaVeiWFynG5Fih3GmY0wpJpVJtCl3rGtSzIG3Lffxat8726nXNTdJjnM+doAZo9cgSVQxpZR0BzO35G6i9Wc32cppkpz+pEPUhhU2hSAaPPIG1vOePW1xIYVvQqA7zJV2GM0Rj+7nvQcM1z6wfPRpsDSVCYRETWJKkq3W/+Q=="
	//response["expires_in"] = 3600

	fmt.Printf("Sending response to Alexa account linking: %+v\n", response)

	ctx.JSON(http.StatusOK, response)
}

func readAPIKey(ctx *gin.Context) string {

	authorization := ctx.Request.Header.Get("Authorization")
	encodedString := strings.Split(authorization, " ")[1]
	decoded, err := base64.StdEncoding.DecodeString(encodedString)
	if err != nil {
		fmt.Println("Error decoding base64 string: ", err.Error())
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return ""
	}
	apiKey := strings.Split(string(decoded), ":")[1]

	return apiKey
}

func dumpRequest(ctx *gin.Context) {

	data, err := httputil.DumpRequest(ctx.Request, true)
	if err != nil {
		fmt.Println("Failed to dump the request: ", err.Error())
		return
	}

	fmt.Println(string(data))
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
		count, _ := echoRequest.GetSlotValue("Count")
		item, _ := echoRequest.GetSlotValue("Item")
		destinationClass, _ := echoRequest.GetSlotValue("Destination")
		output := fmt.Sprintf("Transferring %s of your %s items to your %s", count, item, destinationClass)
		response = alexa.NewEchoResponse()
		response.OutputSpeech(output)
	} else {
		response = alexa.NewEchoResponse()
		response.OutputSpeech("Sorry Guardian, I did not understand your request.")
	}

	bytes, err := response.String()
	if err != nil {
		fmt.Println("Failed to convert EchoResponse to a byte slice.")
		return
	}

	//w.Write(bytes)
	ctx.String(http.StatusOK, string(bytes))
}
