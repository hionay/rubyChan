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

	"github.com/hionay/rubyChan/state"
)

type WeatherCmd struct {
	WeatherAPIKey string
	Store         *state.Namespace
}

func (*WeatherCmd) Name() string      { return "weather" }
func (*WeatherCmd) Aliases() []string { return []string{"w"} }
func (*WeatherCmd) Usage() string {
	return "!weather [location] — Show current weather for [location], or last used by you\n!weather forecast [location] — Show 3-day forecast"
}

func (wc *WeatherCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	user := evt.Sender
	room := evt.RoomID

	forecast := len(args) > 0 && args[0] == "forecast"
	if forecast {
		args = args[1:]
	}

	var loc string
	var err error
	key := room.String() + "|" + user.String()
	if len(args) == 0 {
		loc, err = wc.Store.GetString(key)
		if err != nil {
			cli.SendText(ctx, room, fmt.Sprintf("error retrieving last location: %v", err))
			return
		}
		if loc == "" {
			cli.SendText(ctx, room, "Usage: "+wc.Usage())
			return
		}
	} else {
		loc = strings.Join(args, " ")
	}

	var reply string
	if forecast {
		reply, err = getForecast(wc.WeatherAPIKey, loc)
	} else {
		reply, err = getWeatherOfLocation(wc.WeatherAPIKey, loc)
	}
	if err != nil {
		cli.SendText(ctx, room, fmt.Sprintf("error: %v", err))
		return
	}

	if len(args) > 0 {
		if err := wc.Store.PutString(key, loc); err != nil {
			cli.SendText(ctx, room, fmt.Sprintf("error saving location: %v", err))
			return
		}
	}

	cli.SendText(ctx, room, reply)
}

func getWeatherOfLocation(apiKey, location string) (string, error) {
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
			LastUpdated string  `json:"last_updated"`
			TempC       float64 `json:"temp_c"`
			Condition   struct {
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
		"Weather in %s, %s, %s: %.1f°C, feels like %.1f°C, %s, humidity %d%%, wind %.1f kph (Last updated: %s)",
		wr.Location.Name,
		wr.Location.Region,
		wr.Location.Country,
		wr.Current.TempC,
		wr.Current.FeelsLikeC,
		wr.Current.Condition.Text,
		wr.Current.Humidity,
		wr.Current.WindKph,
		wr.Current.LastUpdated,
	), nil
}

func getForecast(apiKey, location string) (string, error) {
	endpoint := fmt.Sprintf(
		"https://api.weatherapi.com/v1/forecast.json?key=%s&q=%s&days=3&aqi=no&alerts=no",
		apiKey,
		url.QueryEscape(location),
	)

	resp, err := http.Get(endpoint)
	if err != nil {
		return "", fmt.Errorf("failed to fetch forecast: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("weather API returned status %d", resp.StatusCode)
	}

	var fr struct {
		Location struct {
			Name    string `json:"name"`
			Region  string `json:"region"`
			Country string `json:"country"`
		} `json:"location"`
		Forecast struct {
			ForecastDay []struct {
				Date string `json:"date"`
				Day  struct {
					MaxTempC        float64 `json:"maxtemp_c"`
					MinTempC        float64 `json:"mintemp_c"`
					MaxWindKph      float64 `json:"maxwind_kph"`
					TotalPrecipMM   float64 `json:"totalprecip_mm"`
					TotalSnowCM     float64 `json:"totalsnow_cm"`
					AvgHumidity     int     `json:"avghumidity"`
					DailyChanceRain int     `json:"daily_chance_of_rain"`
					DailyChanceSnow int     `json:"daily_chance_of_snow"`
					Condition       struct {
						Text string `json:"text"`
					} `json:"condition"`
				} `json:"day"`
			} `json:"forecastday"`
		} `json:"forecast"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fr); err != nil {
		return "", fmt.Errorf("failed to parse forecast response: %w", err)
	}

	if fr.Location.Name == "" {
		return "Location not found.", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "3-day forecast for %s, %s, %s:\n", fr.Location.Name, fr.Location.Region, fr.Location.Country)
	for _, d := range fr.Forecast.ForecastDay {
		line := fmt.Sprintf("  %s: %.0f°C / %.0f°C, %s | Rain %d%% (%.1fmm) | Wind max %.0fkph | Humidity %d%%",
			d.Date,
			d.Day.MaxTempC, d.Day.MinTempC,
			d.Day.Condition.Text,
			d.Day.DailyChanceRain, d.Day.TotalPrecipMM,
			d.Day.MaxWindKph,
			d.Day.AvgHumidity,
		)
		if d.Day.TotalSnowCM > 0 {
			line += fmt.Sprintf(" | Snow %d%% (%.1fcm)", d.Day.DailyChanceSnow, d.Day.TotalSnowCM)
		}
		sb.WriteString(line + "\n")
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}
