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

	"github.com/gin-gonic/gin"
)

type BungieAPIResponse struct {
	Response        *Response   `json:"Response"`
	ErrorCode       int         `json:"ErrorCode"`
	ThrottleSeconds int         `json:"ThrottleSeconds"`
	ErrorStatus     string      `json:"ErrorStatus"`
	Message         string      `json:"Message"`
	MessageData     interface{} `json:"MessageData"`
}

type Response struct {
	AccessToken  *Token `json:"accessToken"`
	RefreshToken *Token `json:"refreshToken"`
	Scope        int    `json:"scope"`
}

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

	fmt.Println(fmt.Sprintf("Start listening on port(%s)", port))

	router.Run(":" + port)
}

func AuthGetHandler(ctx *gin.Context) {
	dumpRequest(ctx)

	redirectURI = ctx.Query("redirect_uri")
	state = ctx.Query("state")

	ctx.Redirect(http.StatusFound, "https://www.bungie.net/en/Application/Authorize/2579")
}

func AuthCodeResponseHandler(ctx *gin.Context) {
	dumpRequest(ctx)
	uri := fmt.Sprintf("%s?state=%s&code=%s", redirectURI, state, ctx.Query("code"))

	fmt.Println("Sending redirection to: ", uri)
	ctx.Redirect(http.StatusFound, uri)
}

func TokenHandler(ctx *gin.Context) {
	dumpRequest(ctx)

	ctx.Request.ParseForm()

	getAccessToken(ctx)
}

func getAccessToken(ctx *gin.Context) {

	uri := "https://www.bungie.net/Platform/App/GetAccessTokensFromCode/"
	apiKey := readAPIKey(ctx)

	bodyJson := make(map[string]string)
	bodyJson["code"] = ctx.Request.Form.Get("code")
	body, err := json.Marshal(bodyJson)
	if err != nil {
		fmt.Println("Failed to marshal JSON: ", err.Error())
		return
	}

	client := http.Client{}
	req, err := http.NewRequest("POST", uri, strings.NewReader(string(body)))
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
