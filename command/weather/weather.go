package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"

	"github.com/hionay/rubyChan/state"
)

// Nominatim's usage policy requires an identifying User-Agent.
const userAgent = "rubyChan-matrix-bot/1.0 (https://github.com/hionay/rubyChan)"

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Nominatim's usage policy caps requests at one per second.
const nominatimInterval = time.Second

var (
	nominatimMu   sync.Mutex
	nominatimLast time.Time
)

// nominatimWait blocks until the next Nominatim request is allowed. Callers
// queue behind each other, so a burst of lookups is paced rather than refused.
func nominatimWait(ctx context.Context) error {
	nominatimMu.Lock()
	defer nominatimMu.Unlock()

	if d := nominatimInterval - time.Since(nominatimLast); d > 0 {
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	nominatimLast = time.Now()
	return nil
}

type WeatherCmd struct {
	Store *state.Namespace
}

func (*WeatherCmd) Name() string      { return "weather" }
func (*WeatherCmd) Aliases() []string { return []string{"w", "wf"} }
func (*WeatherCmd) Usage() string {
	return "!weather [location] — Show current weather for [location], or last used by you\n!weather forecast [location] — Show 3-day forecast\n!wf [location] — Alias for !weather forecast"
}

func (wc *WeatherCmd) Execute(ctx context.Context, cli *mautrix.Client, evt *event.Event, args []string) {
	user := evt.Sender
	room := evt.RoomID

	raw := strings.TrimSpace(evt.Content.AsMessage().Body)
	forecast := len(args) > 0 && (args[0] == "forecast" || args[0] == "f")
	if !forecast && (raw == "!wf" || strings.HasPrefix(raw, "!wf ")) {
		forecast = true
	}
	if forecast && len(args) > 0 && (args[0] == "forecast" || args[0] == "f") {
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

	geo, err := geocode(ctx, loc)
	if err != nil {
		cli.SendText(ctx, room, fmt.Sprintf("error: %v", err))
		return
	}
	if geo == nil {
		cli.SendText(ctx, room, fmt.Sprintf("Location not found: %s", loc))
		return
	}

	var reply string
	if forecast {
		reply, err = getForecast(ctx, geo)
	} else {
		reply, err = getWeatherOfLocation(ctx, geo)
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

type geoResult struct {
	Name    string
	Region  string
	Country string
	Lat     float64
	Lon     float64
}

func (g *geoResult) label() string {
	parts := make([]string, 0, 3)
	for _, p := range []string{g.Name, g.Region, g.Country} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, ", ")
}

// geocode resolves a free-text location. Open-Meteo's geocoder gives clean
// structured names but misses ASCII-folded Turkish names (e.g. "Kagithane"),
// so Nominatim covers the gap. A nil result with a nil error means not found.
func geocode(ctx context.Context, location string) (*geoResult, error) {
	g, err := geocodeOpenMeteo(ctx, location)
	if g != nil {
		return g, nil
	}
	g2, err2 := geocodeNominatim(ctx, location)
	if g2 != nil {
		return g2, nil
	}
	if err != nil {
		return nil, err
	}
	return nil, err2
}

func geocodeOpenMeteo(ctx context.Context, location string) (*geoResult, error) {
	q := url.Values{
		"name":     {location},
		"count":    {"1"},
		"language": {"en"},
		"format":   {"json"},
	}
	var res struct {
		Results []struct {
			Name      string  `json:"name"`
			Admin1    string  `json:"admin1"`
			Country   string  `json:"country"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"results"`
	}
	if err := getJSON(ctx, "https://geocoding-api.open-meteo.com/v1/search?"+q.Encode(), &res); err != nil {
		return nil, fmt.Errorf("failed to geocode location: %w", err)
	}
	if len(res.Results) == 0 {
		return nil, nil
	}
	r := res.Results[0]
	return &geoResult{Name: r.Name, Region: r.Admin1, Country: r.Country, Lat: r.Latitude, Lon: r.Longitude}, nil
}

func geocodeNominatim(ctx context.Context, location string) (*geoResult, error) {
	q := url.Values{
		"q":              {location},
		"format":         {"jsonv2"},
		"limit":          {"1"},
		"addressdetails": {"1"},
		"featureType":    {"settlement"},
	}
	var res []struct {
		Name    string `json:"name"`
		Lat     string `json:"lat"`
		Lon     string `json:"lon"`
		Address struct {
			State    string `json:"state"`
			Province string `json:"province"`
			Region   string `json:"region"`
			Country  string `json:"country"`
		} `json:"address"`
	}
	if err := nominatimWait(ctx); err != nil {
		return nil, err
	}
	if err := getJSON(ctx, "https://nominatim.openstreetmap.org/search?"+q.Encode(), &res); err != nil {
		return nil, fmt.Errorf("failed to geocode location: %w", err)
	}
	if len(res) == 0 {
		return nil, nil
	}
	r := res[0]
	lat, err := strconv.ParseFloat(r.Lat, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse latitude %q: %w", r.Lat, err)
	}
	lon, err := strconv.ParseFloat(r.Lon, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse longitude %q: %w", r.Lon, err)
	}
	region := r.Address.State
	if region == "" {
		region = r.Address.Province
	}
	if region == "" {
		region = r.Address.Region
	}
	return &geoResult{Name: r.Name, Region: region, Country: r.Address.Country, Lat: lat, Lon: lon}, nil
}

func getWeatherOfLocation(ctx context.Context, geo *geoResult) (string, error) {
	q := url.Values{
		"latitude":        {fmt.Sprintf("%f", geo.Lat)},
		"longitude":       {fmt.Sprintf("%f", geo.Lon)},
		"current":         {"temperature_2m,apparent_temperature,relative_humidity_2m,wind_speed_10m,weather_code"},
		"wind_speed_unit": {"kmh"},
		"timezone":        {"auto"},
	}
	var res struct {
		Current struct {
			Time        string  `json:"time"`
			TempC       float64 `json:"temperature_2m"`
			FeelsLikeC  float64 `json:"apparent_temperature"`
			Humidity    int     `json:"relative_humidity_2m"`
			WindKph     float64 `json:"wind_speed_10m"`
			WeatherCode int     `json:"weather_code"`
		} `json:"current"`
	}
	if err := getJSON(ctx, "https://api.open-meteo.com/v1/forecast?"+q.Encode(), &res); err != nil {
		return "", fmt.Errorf("failed to fetch weather: %w", err)
	}

	c := res.Current
	return fmt.Sprintf(
		"Weather in %s: %.1f°C, feels like %.1f°C, %s, humidity %d%%, wind %.1f kph (Last updated: %s)",
		geo.label(),
		c.TempC,
		c.FeelsLikeC,
		wmoText(c.WeatherCode),
		c.Humidity,
		c.WindKph,
		strings.Replace(c.Time, "T", " ", 1),
	), nil
}

func getForecast(ctx context.Context, geo *geoResult) (string, error) {
	q := url.Values{
		"latitude":        {fmt.Sprintf("%f", geo.Lat)},
		"longitude":       {fmt.Sprintf("%f", geo.Lon)},
		"daily":           {"weather_code,temperature_2m_max,temperature_2m_min,precipitation_sum,precipitation_probability_max,wind_speed_10m_max,snowfall_sum,relative_humidity_2m_mean"},
		"wind_speed_unit": {"kmh"},
		"forecast_days":   {"3"},
		"timezone":        {"auto"},
	}
	var res struct {
		Daily struct {
			Time        []string  `json:"time"`
			WeatherCode []int     `json:"weather_code"`
			MaxTempC    []float64 `json:"temperature_2m_max"`
			MinTempC    []float64 `json:"temperature_2m_min"`
			PrecipMM    []float64 `json:"precipitation_sum"`
			ChanceRain  []int     `json:"precipitation_probability_max"`
			MaxWindKph  []float64 `json:"wind_speed_10m_max"`
			SnowCM      []float64 `json:"snowfall_sum"`
			Humidity    []int     `json:"relative_humidity_2m_mean"`
		} `json:"daily"`
	}
	if err := getJSON(ctx, "https://api.open-meteo.com/v1/forecast?"+q.Encode(), &res); err != nil {
		return "", fmt.Errorf("failed to fetch forecast: %w", err)
	}

	d := res.Daily
	if min(len(d.WeatherCode), len(d.MaxTempC), len(d.MinTempC), len(d.PrecipMM),
		len(d.ChanceRain), len(d.MaxWindKph), len(d.SnowCM), len(d.Humidity)) < len(d.Time) {
		return "", fmt.Errorf("incomplete forecast response")
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "3-day forecast for %s:\n", geo.label())
	for i := range d.Time {
		line := fmt.Sprintf("  %s: %.0f°C / %.0f°C, %s | Rain %d%% (%.1fmm) | Wind max %.0fkph | Humidity %d%%",
			d.Time[i],
			d.MaxTempC[i], d.MinTempC[i],
			wmoText(d.WeatherCode[i]),
			d.ChanceRain[i], d.PrecipMM[i],
			d.MaxWindKph[i],
			d.Humidity[i],
		)
		if d.SnowCM[i] > 0 {
			line += fmt.Sprintf(" | Snow %.1fcm", d.SnowCM[i])
		}
		fmt.Fprintln(&sb, line)
	}
	return strings.TrimRight(sb.String(), "\n"), nil
}

func getJSON(ctx context.Context, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// https://open-meteo.com/en/docs — WMO weather interpretation codes.
var wmoCodes = map[int]string{
	0:  "Clear",
	1:  "Mainly clear",
	2:  "Partly cloudy",
	3:  "Overcast",
	45: "Fog",
	48: "Depositing rime fog",
	51: "Light drizzle",
	53: "Moderate drizzle",
	55: "Dense drizzle",
	56: "Light freezing drizzle",
	57: "Dense freezing drizzle",
	61: "Slight rain",
	63: "Moderate rain",
	65: "Heavy rain",
	66: "Light freezing rain",
	67: "Heavy freezing rain",
	71: "Slight snow fall",
	73: "Moderate snow fall",
	75: "Heavy snow fall",
	77: "Snow grains",
	80: "Slight rain showers",
	81: "Moderate rain showers",
	82: "Violent rain showers",
	85: "Slight snow showers",
	86: "Heavy snow showers",
	95: "Thunderstorm",
	96: "Thunderstorm with slight hail",
	99: "Thunderstorm with heavy hail",
}

func wmoText(code int) string {
	if text, ok := wmoCodes[code]; ok {
		return text
	}
	return fmt.Sprintf("Unknown (%d)", code)
}
