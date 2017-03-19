package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/gin-gonic/gin"
)

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
	router.POST("/auth", AuthPostHandler)

	router.GET("auth-code-response", AuthCodeResponseHandler)

	router.GET("tokens", TokenHandler)
	router.POST("tokens", PostTokenHandler)

	fmt.Println(fmt.Sprintf("Start listening on port(%s)", port))

	router.Run(":" + port)
}

func AuthGetHandler(ctx *gin.Context) {
	dumpRequest(ctx)
	SharedAuthHandler(ctx)
}

func AuthPostHandler(ctx *gin.Context) {
	dumpRequest(ctx)
	SharedAuthHandler(ctx)
}

func SharedAuthHandler(ctx *gin.Context) {
	ctx.Redirect(http.StatusSeeOther, "https://www.bungie.net/en/Application/Authorize/2579")
}

func AuthCodeResponseHandler(ctx *gin.Context) {
	dumpRequest(ctx)
	ctx.Redirect(http.StatusFound, "https://layla.amazon.com/api/skill/link/M2H4SPMARTTC13")
}

func TokenHandler(ctx *gin.Context) {
	dumpRequest(ctx)
}

func PostTokenHandler(ctx *gin.Context) {
	dumpRequest(ctx)
}

func dumpRequest(ctx *gin.Context) {

	data, err := httputil.DumpRequest(ctx.Request, true)
	if err != nil {
		fmt.Println("Failed to dump the request: ", err.Error())
		return
	}

	fmt.Println(string(data))
}
