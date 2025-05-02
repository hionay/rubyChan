package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
)

type WeatherCmd struct {
	WeatherAPIKey string
}

func (*WeatherCmd) Name() string      { return "weather" }
func (*WeatherCmd) Aliases() []string { return []string{"w"} }
func (*WeatherCmd) Usage() string {
	return "!weather <location> - Show current weather for <location>"
}

func (wc *WeatherCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	if len(args) == 0 {
		cli.SendText(ctx, evt.RoomID, "Usage: "+wc.Usage())
		return
	}
	loc := strings.Join(args, " ")
	reply, err := getWeatherOfLocation(wc.WeatherAPIKey, loc)
	if err != nil {
		cli.SendText(ctx, evt.RoomID, fmt.Sprintf("️error: %v", err))
		return
	}
	cli.SendText(ctx, evt.RoomID, reply)
}

func getWeatherOfLocation(apiKey string, location string) (string, error) {
	endpoint := fmt.Sprintf(
		"https://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no",
		apiKey,
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
