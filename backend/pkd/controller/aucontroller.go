package controller

import (
	"angular-and-go/pkd/appuser"
	"angular-and-go/pkd/appuser/aumodel"
	aufile "angular-and-go/pkd/appuser/file"
	aubody "angular-and-go/pkd/controller/aumodel"
	token "angular-and-go/pkd/token"
	"fmt"
	"log"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
)

func getRefreshToken(c *gin.Context) {
	status := http.StatusUnauthorized
	message := "Invalid"
	result := ""
	userName, exits := c.Get("user")
	roles, exists2 := c.Get("roles")
	if exits && exists2 {
		//jwt token creation
		var err error
		result, err = token.CreateToken(token.TokenUser{Username: userName.(string), Roles: []string{roles.(string)}})
		if err != nil {
			log.Printf("Failed to create jwt token: %v\n", err)
		} else {
			status = http.StatusOK
			message = ""
		}
	}
	c.JSON(status, aubody.AppUserResponse{Token: result, Message: message})
}

func getLocation(c *gin.Context) {
	locationStr := c.Query("location")
	postCodeLocations := appuser.FindLocation(locationStr)
	//log.Printf("Locations: %v", postCodeLocations)
	myPostCodeLocations := mapToPostCodeLocation(postCodeLocations)
	c.JSON(http.StatusOK, myPostCodeLocations)
}

func mapToPostCodeLocation(postCodeLocations []aumodel.PostCodeLocation) []aubody.CodeLocationResponse {
	result := []aubody.CodeLocationResponse{}
	for _, postCodeLocation := range postCodeLocations {
		if !math.IsNaN(postCodeLocation.CenterLatitude) && !math.IsNaN(postCodeLocation.CenterLongitude) && !math.IsNaN(float64(postCodeLocation.SquareKM)) {
			myPostCodeLocation := aubody.CodeLocationResponse{
				Longitude:  postCodeLocation.CenterLongitude,
				Latitude:   postCodeLocation.CenterLatitude,
				Label:      postCodeLocation.Label,
				PostCode:   postCodeLocation.PostCode,
				SquareKM:   postCodeLocation.SquareKM,
				Population: postCodeLocation.Population,
			}
			result = append(result, myPostCodeLocation)
		}
	}
	return result
}

func getPostCodeCoordinates(c *gin.Context) {
	filePath := c.Query("filename")
	aufile.UpdatePostCodeCoordinates(filePath)
}

func postSignin(c *gin.Context) {
	//jsonData, err := ioutil.ReadAll(c.Request.Body)
	//fmt.Printf("Json: %v, Err: %v", string(jsonData), err)
	var appUserRequest aubody.AppUserRequest
	if err := c.Bind(&appUserRequest); err != nil {
		log.Printf("postSingin: %v", err.Error())
	}
	myAppUser := appuser.AppUserIn{Username: appUserRequest.Username, Password: appUserRequest.Password, Uuid: ""}
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
	if err := c.Bind(&appUserRequest); err != nil {
		log.Printf("postLogin: %v", err.Error())
	}
	myAppUser := appuser.AppUserIn{Username: appUserRequest.Username, Password: appUserRequest.Password, Uuid: ""}
	result, status, userLongitude, userLatitude, searchRadius, targetE5, targetE10, targetDiesel := appuser.Login(myAppUser)
	var message = ""
	if status != http.StatusOK {
		message = "Login failed."
	}
	appAuResponse := aubody.AppUserResponse{Token: result, Message: message, Longitude: userLongitude, Latitude: userLatitude,
		SearchRadius: searchRadius, TargetE5: fmt.Sprintf("%v", (float64(targetE5) / 1000)), TargetE10: fmt.Sprintf("%v", (float64(targetE10) / 1000)), TargetDiesel: fmt.Sprintf("%v", (float64(targetDiesel) / 1000))}
	c.JSON(status, appAuResponse)
}

func postUserLocationRadius(c *gin.Context) {
	var appUserRequest aubody.AppUserRequest
	if err := c.Bind(&appUserRequest); err != nil {
		log.Printf("putUserLocationRadius: %v", err.Error())
	}
	myAppUser := appuser.AppUserIn{Username: appUserRequest.Username, Uuid: "", Longitude: appUserRequest.Longitude, Latitude: appUserRequest.Latitude, SearchRadius: appUserRequest.SearchRadius}
	result := appuser.StoreLocationAndRadius(myAppUser)
	httpResult := http.StatusOK
	message := "Ok"
	if result != appuser.Ok {
		httpResult = http.StatusBadRequest
		message = "Invalid"
	}
	c.JSON(httpResult, aubody.CodeLocationResponse{Message: message, Label: "", Longitude: appUserRequest.Longitude, Latitude: appUserRequest.Latitude, PostCode: 0, SquareKM: 0, Population: 0})
}

func postTargetPrices(c *gin.Context) {
	var appUserRequest aubody.AppUserRequest
	if err := c.Bind(&appUserRequest); err != nil {
		log.Printf("putUserLocationRadius: %v", err.Error())
	}
	myTargetPrices := appuser.AppTargetIn{Username: appUserRequest.Username, TargetDiesel: appUserRequest.TargetDiesel, TargetE10: appUserRequest.TargetE10, TargetE5: appUserRequest.TargetE5}
	result := appuser.StoreTargetPrices(myTargetPrices)
	httpResult := http.StatusOK
	message := "Ok"
	if result != appuser.Ok {
		httpResult = http.StatusBadRequest
		message = "Invalid"
	}
	c.JSON(httpResult, aubody.TargetPricesResponse{Message: message, TargetDiesel: appUserRequest.TargetDiesel, TargetE10: appUserRequest.TargetE10, TargetE5: appUserRequest.TargetE5})
}
