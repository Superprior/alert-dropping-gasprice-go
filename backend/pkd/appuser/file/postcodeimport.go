/*
  - Copyright 2022 Sven Loesekann
    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
*/
package aufile

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"os"
	appuser "react-and-go/pkd/appuser"
	"strings"
)

type coordinateTuple [2]float64

type coordinateList []coordinateTuple

type plzPolygon struct {
	Typestr     string `json:"type"`
	Coordinates []coordinateList
}

type plzProperties struct {
	Plz        int32   `json:"plz,string"`
	Label      string  `json:"note"`
	Population int32   `json:"einwohner"`
	SquareKM   float32 `json:"qkm"`
}

type plzContainer struct {
	Typestr    string        `json:"type"`
	Properties plzProperties `json:"properties"`
	Geometry   plzPolygon    `json:"geometry"`
}

func UpdatePostCodeCoordinates(fileName string) {
	filePath := strings.TrimSpace(os.Getenv("PLZ_IMPORT_PATH"))
	log.Printf("File: %v%v", filePath, fileName)
	file, err := os.Open(fmt.Sprintf("%v%v", filePath, strings.TrimSpace(fileName)))
	defer file.Close()
	if err != nil {
		log.Printf("Failed to open file: %v, %v\n", fmt.Sprintf("%v%v", filePath, strings.TrimSpace(fileName)), err.Error())
		return
	}
	gzReader, err := gzip.NewReader(bufio.NewReader(file))
	if err != nil {
		log.Printf("Failed to create buffered gzip reader: %v, %v\n", fmt.Sprintf("%v%v", filePath, strings.TrimSpace(fileName)), err.Error())
		return
	}
	defer gzReader.Close()

	jsonDecoder := json.NewDecoder(gzReader)
	plzContainerNumber := 0
	result := []appuser.PostCodeData{}

	jsonDecoder.Token()
	for jsonDecoder.More() {
		myPlzContainer := plzContainer{}
		jsonDecoder.Decode(&myPlzContainer)
		plzContainerNumber++
		myPostCode := createPostCode(&myPlzContainer)
		result = append(result, myPostCode)
		//log.Printf("PostCode: %v\n", myPostCode)
	}
	jsonDecoder.Token()
	//log.Printf("Number of postcodes: %v\n", plzContainerNumber)
	appuser.ImportPostCodeData(result)
}

func createPostCode(plzContainer *plzContainer) appuser.PostCodeData {
	postCodeData := appuser.PostCodeData{}
	postCodeData.Label = plzContainer.Properties.Label
	postCodeData.PostCode = plzContainer.Properties.Plz
	postCodeData.SquareKM = plzContainer.Properties.SquareKM
	postCodeData.Population = plzContainer.Properties.Population
	postCodeData.CenterLongitude, postCodeData.CenterLatitude = calcCentroid(plzContainer)
	return postCodeData
}

func calcCentroid(plzContainer *plzContainer) (float64, float64) {
	polygonArea := calcPolygonArea(plzContainer)
	//log.Printf("PolygonArea: %v", polygonArea)
	coordinateLists := plzContainer.Geometry.Coordinates
	centerLongitude := 0.0
	centerLatitude := 0.0
	for _, coordinateTuples := range coordinateLists {
		for index, coordinateTuple := range coordinateTuples {
			if index >= len(coordinateTuples)-1 {
				continue
			}
			centerLongitude += (coordinateTuple[0] + coordinateTuples[index+1][0]) * (coordinateTuple[0]*coordinateTuples[index+1][1] - coordinateTuples[index+1][0]*coordinateTuple[1])
			centerLatitude += (coordinateTuple[1] + coordinateTuples[index+1][1]) * (coordinateTuple[0]*coordinateTuples[index+1][1] - coordinateTuples[index+1][0]*coordinateTuple[1])
		}
	}
	centerLongitude = centerLongitude / (6 * polygonArea)
	centerLatitude = centerLatitude / (6 * polygonArea)
	return centerLongitude, centerLatitude
}

func calcPolygonArea(plzContainer *plzContainer) float64 {
	coordinateLists := plzContainer.Geometry.Coordinates
	polygonArea := 0.0
	for _, coordinateTuples := range coordinateLists {
		for index, coordinateTuple := range coordinateTuples {
			if index >= len(coordinateTuples)-1 {
				continue
			}
			polygonArea += coordinateTuple[0]*coordinateTuples[index+1][1] - coordinateTuples[index+1][0]*coordinateTuple[1]
		}
	}
	polygonArea = polygonArea / 2
	return polygonArea
}
