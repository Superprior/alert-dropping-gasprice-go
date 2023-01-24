package contr

import (
	"angular-and-go/pkd/appuser"
	aubody "angular-and-go/pkd/contr/aumodel"
	gsclient "angular-and-go/pkd/contr/client"
	gsbody "angular-and-go/pkd/contr/gsmodel"
	"angular-and-go/pkd/gasstation"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Start() {
	router := gin.Default()
	router.POST("/appuser/signin", postSignin)
	router.POST("/appuser/login", postLogin)
	router.GET("/clienttest", gsclient.UpdateGasStations)
	router.GET("/gasprice/:id", getGasPriceByGasStationId)
	router.GET("/gasstation/:id", getGasStationById)
	router.POST("/gasstation/search/place", searchGasStationPlace)
	router.POST("/gasstation/search/location", searchGasStationLocation)
	router.PUT("/posts/:id", postsUpdate)
	router.DELETE("/posts/:id", postsDelete)
	router.Static("/static", "./static")
	router.NoRoute(func(c *gin.Context) { c.Redirect(302, "/static") })
	router.Run() // listen and serve on 0.0.0.0:3000
}

func postSignin(c *gin.Context) {
	var appUserRequest aubody.AppUserRequest
	c.Bind(&appUserRequest)
	myAppUser := appuser.AppUserIn{Username: appUserRequest.Username, Password: appUserRequest.Password, Latitude: appUserRequest.Latitude, Uuid: "", Longitude: appUserRequest.Longitude}
	result := appuser.Signin(myAppUser)
	httpResult := http.StatusNotAcceptable
	message := ""
	if result == appuser.Ok {
		httpResult = http.StatusAccepted
	} else if result == appuser.UsernameTaken {
		message = "Username not available."
	}
	c.JSON(httpResult, aubody.AppUserResponse{Token: "", Message: message})
}

func postLogin(c *gin.Context) {
	var appUserRequest aubody.AppUserRequest
	c.Bind(&appUserRequest)
	myAppUser := appuser.AppUserIn{Username: appUserRequest.Username, Password: appUserRequest.Password, Latitude: appUserRequest.Latitude, Uuid: "", Longitude: appUserRequest.Longitude}
	result, status := appuser.Login(myAppUser)
	var message = ""
	if status != http.StatusOK {
		message = "Login failed."
	}
	appAuResponse := aubody.AppUserResponse{Token: result, Message: message}
	c.JSON(status, appAuResponse)
}

func getGasPriceByGasStationId(c *gin.Context) {
	gasstationId := c.Params.ByName("id")
	gsEntity := gasstation.FindPricesByStid(gasstationId)
	c.JSON(http.StatusOK, gsEntity)
}

func getGasStationById(c *gin.Context) {
	gasstationId := c.Params.ByName("id")
	gsEntity := gasstation.FindById(gasstationId)
	c.JSON(http.StatusOK, gsEntity)
}

func searchGasStationPlace(c *gin.Context) {
	var searchPlaceBody gsbody.SearchPlaceBody
	c.Bind(&searchPlaceBody)
	gsEntity := gasstation.FindBySearchPlace(searchPlaceBody)
	c.JSON(http.StatusOK, gsEntity)
}

func searchGasStationLocation(c *gin.Context) {
	//jsonData, err := ioutil.ReadAll(c.Request.Body)
	//fmt.Printf("Json: %v, Err: %v", string(jsonData), err)
	var searchLocationBody gsbody.SearchLocation
	c.Bind(&searchLocationBody)
	//fmt.Printf("Lat: %v, Lng: %v\n", searchLocationBody.Latitude, searchLocationBody.Longitude)
	gsEntity := gasstation.FindBySearchLocation(searchLocationBody)
	c.JSON(http.StatusOK, gsEntity)
}

func postsUpdate(c *gin.Context) {

}

func postsDelete(c *gin.Context) {

}
