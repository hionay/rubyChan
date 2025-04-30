package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func searchWeather(cfg *Config, location string) (string, error) {
	if cfg.WeatherAPIKey == "" {
		return "", fmt.Errorf("WEATHER_API_KEY not set")
	}

	endpoint := fmt.Sprintf(
		"https://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no",
		cfg.WeatherAPIKey,
		url.QueryEscape(location),
	)

	resp, err := http.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to fetch weather: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("weather API returned status %d", resp.StatusCode)
	}

	var wr struct {
		Location struct {
			Name    string `json:"name"`
			Region  string `json:"region"`
			Country string `json:"country"`
		} `json:"location"`
		Current struct {
			TempC     float64 `json:"temp_c"`
			Condition struct {
				Text string `json:"text"`
			} `json:"condition"`
			Humidity   int     `json:"humidity"`
			FeelsLikeC float64 `json:"feelslike_c"`
			WindKph    float64 `json:"wind_kph"`
		} `json:"current"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wr); err != nil {
		return "", fmt.Errorf("failed to parse weather response: %w", err)
	}

	if wr.Location.Name == "" {
		return "Location not found.", nil
	}

	return fmt.Sprintf(
		"Weather in %s, %s, %s: %.1f°C, feels like %.1f°C, %s, humidity %d%%, wind %.1f kph",
		wr.Location.Name,
		wr.Location.Region,
		wr.Location.Country,
		wr.Current.TempC,
		wr.Current.FeelsLikeC,
		wr.Current.Condition.Text,
		wr.Current.Humidity,
		wr.Current.WindKph,
	), nil
}
