package appuser

import (
	"fmt"
	"log"
	"net/http"
	"react-and-go/pkd/appuser/aumodel"
	"react-and-go/pkd/database"
	token "react-and-go/pkd/token"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AppUserIn struct {
	Username     string
	Password     string
	Uuid         string
	Latitude     float64
	Longitude    float64
	SearchRadius float64
}

type AppTargetIn struct {
	Username     string
	TargetDiesel string
	TargetE10    string
	TargetE5     string
}

type DbResult int

type PostCodeData struct {
	Label           string
	PostCode        int32
	Population      int32
	SquareKM        float32
	CenterLongitude float64
	CenterLatitude  float64
}

const (
	Ok DbResult = iota
	UsernameTaken
	Invalid
	Failed
)

func StoreUserLogout(username string, uuid string) []token.LoggedOutUserOut {
	database.DB.Delete("last_logout < ?", time.Now().Add(-4*time.Minute))
	loggedOutUser := aumodel.LoggedOutUser{Username: username, Uuid: uuid, LastLogout: time.Now()}
	database.DB.Save(loggedOutUser)
	var loggedOutUsers []aumodel.LoggedOutUser
	database.DB.Find(&loggedOutUsers)
	var results []token.LoggedOutUserOut
	for _, myLoggedOutUser := range loggedOutUsers {
		loggedOutUserOut := token.LoggedOutUserOut{Username: myLoggedOutUser.Username, Uuid: myLoggedOutUser.Uuid, LastLogout: myLoggedOutUser.LastLogout}
		results = append(results, loggedOutUserOut)
	}
	return results
}

func FindLocation(locationStr string) []aumodel.PostCodeLocation {
	result := []aumodel.PostCodeLocation{}
	database.DB.Where("lower(label) like ?", fmt.Sprintf("%%%v%%", strings.ToLower(strings.TrimSpace(locationStr)))).Limit(20).Find(&result)
	//log.Printf("Select: %v failed. %v", fmt.Sprintf("%%%v%%", strings.ToLower(strings.TrimSpace(locationStr))), err)
	return result
}

func Login(appUserIn AppUserIn) (string, int, float64, float64, float64, int, int, int) {
	result := ""
	status := http.StatusUnauthorized
	//log.Printf("%v", appUserIn.Username)
	var appUser aumodel.AppUser
	if err := database.DB.Where("username = ?", appUserIn.Username).First(&appUser); err.Error != nil {
		log.Printf("User not found: %v error: %v\n", appUserIn.Username, err.Error)
		return result, status, 0.0, 0.0, 0.0, 0, 0, 0
	}
	if err := bcrypt.CompareHashAndPassword([]byte(appUser.Password), []byte(appUserIn.Password)); err != nil {
		log.Printf("Password wrong. Username: %v\n", appUser.Username)
		return result, status, 0.0, 0.0, 0.0, 0, 0, 0
	}
	//jwt token creation
	result, err := token.CreateToken(token.TokenUser{Username: appUser.Username, Roles: []string{"USERS"}})
	if err != nil {
		log.Printf("Failed to create jwt token: %v\n", err)
	} else {
		status = http.StatusOK
	}
	return result, status, appUser.Longitude, appUser.Latitude, appUser.SearchRadius, appUser.TargetE5, appUser.TargetE10, appUser.TargetDiesel
}

func Signin(appUserIn AppUserIn) DbResult {
	var result DbResult = Invalid
	if len(appUserIn.Username) < 4 || len(appUserIn.Password) < 8 {
		return result
	}
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var appUser aumodel.AppUser
		if err := tx.Where("username = ?", appUserIn.Username).First(&appUser); err.Error == nil {
			result = UsernameTaken
			return nil
		}
		appUser.Username = appUserIn.Username
		appUser.Password = string(generatePasswordHash(appUserIn.Password))
		tx.Save(&appUser)
		return nil
	})
	if err != nil {
		result = Failed
	} else if result == Invalid {
		result = Ok
	}
	return result
}

func StoreLocationAndRadius(appUserIn AppUserIn) DbResult {
	result := Invalid
	database.DB.Transaction(func(tx *gorm.DB) error {
		var appUser aumodel.AppUser
		if err := tx.Where("username = ?", appUserIn.Username).Find(&appUser); err.Error == nil {
			appUser.Longitude = appUserIn.Longitude
			appUser.Latitude = appUserIn.Latitude
			appUser.SearchRadius = appUserIn.SearchRadius
			tx.Save(&appUser)
			result = Ok
		}
		return nil
	})
	return result
}

func StoreTargetPrices(appTargetIn AppTargetIn) DbResult {
	result := Invalid
	database.DB.Transaction(func(tx *gorm.DB) error {
		var appUser aumodel.AppUser
		var txError error = nil
		if err := tx.Where("username = ?", appTargetIn.Username).Find(&appUser); err.Error == nil {
			if targetPrice, err := strconv.ParseInt(strings.ReplaceAll(appTargetIn.TargetDiesel, ".", ""), 10, 32); err == nil {
				appUser.TargetDiesel = int(targetPrice)
			} else {
				log.Printf("TargetDiesel: %v\n", appTargetIn.TargetDiesel)
				txError = err
			}
			if targetPrice, err := strconv.ParseInt(strings.ReplaceAll(appTargetIn.TargetE10, ".", ""), 10, 32); err == nil {
				appUser.TargetE10 = int(targetPrice)
			} else {
				log.Printf("TargetE10: %v\n", appTargetIn.TargetE10)
				txError = err
			}
			if targetPrice, err := strconv.ParseInt(strings.ReplaceAll(appTargetIn.TargetE5, ".", ""), 10, 32); err == nil {
				appUser.TargetE5 = int(targetPrice)
			} else {
				log.Printf("TargetE5: %v\n", appTargetIn.TargetE5)
				txError = err
			}
			if txError == nil {
				tx.Save(&appUser)
				result = Ok
			}
		}
		return txError
	})
	return result
}

func ImportPostCodeData(postCodeData []PostCodeData) {
	postCodeLocations := mapToPostCodeLocation(postCodeData)
	var oriPostCodeLocations []aumodel.PostCodeLocation
	database.DB.Find(&oriPostCodeLocations)
	postCodeLocationsMap := make(map[int32]aumodel.PostCodeLocation)
	for _, oriPostCodeLocation := range oriPostCodeLocations {
		postCodeLocationsMap[oriPostCodeLocation.PostCode] = oriPostCodeLocation
	}
	database.DB.Transaction(func(tx *gorm.DB) error {
		for _, postCodeLocation := range postCodeLocations {
			oriPostCodeLocation, exists := postCodeLocationsMap[postCodeLocation.PostCode]
			if exists {
				oriPostCodeLocation.Label = postCodeLocation.Label
				oriPostCodeLocation.PostCode = postCodeLocation.PostCode
				oriPostCodeLocation.Population = postCodeLocation.Population
				oriPostCodeLocation.SquareKM = postCodeLocation.SquareKM
				oriPostCodeLocation.CenterLongitude = postCodeLocation.CenterLongitude
				oriPostCodeLocation.CenterLatitude = postCodeLocation.CenterLatitude
				tx.Save(&oriPostCodeLocation)
			} else {
				tx.Save(&postCodeLocation)
			}
		}
		return nil
	})
	log.Printf("PostCodeLocations saved: %v\n", len(postCodeLocations))
}

func mapToPostCodeLocation(postCodeData []PostCodeData) []aumodel.PostCodeLocation {
	result := []aumodel.PostCodeLocation{}
	for _, myPostCodeData := range postCodeData {
		myPostCodeLocation := aumodel.PostCodeLocation{}
		myPostCodeLocation.Label = myPostCodeData.Label
		myPostCodeLocation.PostCode = myPostCodeData.PostCode
		myPostCodeLocation.Population = myPostCodeData.Population
		myPostCodeLocation.SquareKM = myPostCodeData.SquareKM
		myPostCodeLocation.CenterLongitude = myPostCodeData.CenterLongitude
		myPostCodeLocation.CenterLatitude = myPostCodeData.CenterLatitude
		result = append(result, myPostCodeLocation)
	}
	return result
}

func generatePasswordHash(password string) []byte {
	passwordSlice := []byte(password)
	hashValue, err := bcrypt.GenerateFromPassword(passwordSlice, bcrypt.DefaultCost)
	if err != nil {
		log.Printf("GenerateFromPassword failed: %v\n", err.Error())
	}
	return hashValue
}
