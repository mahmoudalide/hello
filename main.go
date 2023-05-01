package main

import (
	"encoding/json"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
)

// https://howistart.org/posts/go/1/
//
//	type weatherData struct {
//		Name string `json:"name"`
//		Main struct {
//			Kelvin float64 `json:"temp"`
//		} `json:"main"`
//	}
type weatherProvider interface {
	temperature(city string) (float64, error) // in Kelvin, naturally
}

type openWeatherMap struct{}

func (w openWeatherMap) temperature(city string) (float64, error) {
	resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?APPID=<KEY>&q=" + city)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var d struct {
		Main struct {
			Kelvin float64 `json:"temp"`
		} `json:"main"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}
	d.Main.Kelvin = math.Round((d.Main.Kelvin-273.15)*100) / 100
	log.Printf("openWeatherMap: %s: %.2f", city, d.Main.Kelvin)
	return d.Main.Kelvin, nil
}

type weatherApi struct {
	apiKey string //aqi=6140c3fba23b496fac5200704230105
}

func (w weatherApi) temperature(city string) (float64, error) {
	resp, err := http.Get("http://api.weatherapi.com/v1/current.json?key=" + w.apiKey + "&q=" + city + "&aqi=yes")
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var d struct {
		Current struct {
			Celsius float64 `json:"feelslike_c"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}

	//kelvin := d.Current.Celsius + 273.15
	log.Printf("weatherApi: %s: %.2f", city, d.Current.Celsius)
	return d.Current.Celsius, nil
}
func temperature(city string, providers ...weatherProvider) (float64, error) {
	sum := 0.0

	for _, provider := range providers {
		k, err := provider.temperature(city)
		if err != nil {
			return 0, err
		}

		sum += k
	}

	return sum / float64(len(providers)), nil
}

type multiWeatherProvider []weatherProvider

func (w multiWeatherProvider) temperature(city string) (float64, error) {
	sum := 0.0

	for _, provider := range w {
		k, err := provider.temperature(city)
		if err != nil {
			return 0, err
		}

		sum += k
	}

	return sum / float64(len(w)), nil
}
func main() {
	http.HandleFunc("/hello", hello)
	mw := multiWeatherProvider{
		openWeatherMap{},
		weatherApi{apiKey: "<KEY>"},
	}

	http.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()
		city := strings.SplitN(r.URL.Path, "/", 3)[2]

		temp, err := mw.temperature(city)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"city": city,
			"temp": temp,
			"took": time.Since(begin).String(),
		})
	})

	http.ListenAndServe(":8080", nil)
}

func hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello"))
}

// func query(city string) (weatherData, error) {
// 	resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?APPID=fc5b1790843d5d172a7f3a128bfaa5c9&q=" + city)
// 	if err != nil {
// 		return weatherData{}, err
// 	}

// 	defer resp.Body.Close()

// 	var d weatherData

// 	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
// 		return weatherData{}, err
// 	}
// 	d.Main.Kelvin = math.Round((d.Main.Kelvin-273.15)*100) / 100
// 	return d, nil
// }
