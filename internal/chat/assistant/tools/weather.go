package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/openai/openai-go/v2"
)

// SimpleWeatherService provides basic weather functionality
type SimpleWeatherService struct {
	apiKey string
	client *http.Client
}

func NewSimpleWeatherService(apiKey string) *SimpleWeatherService {
	return &SimpleWeatherService{
		apiKey: apiKey,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// WeatherAPIResponse represents the structure of the WeatherAPI.com response
type WeatherAPIResponse struct {
	Location struct {
		Name    string `json:"name"`
		Region  string `json:"region"`
		Country string `json:"country"`
	} `json:"location"`
	Current struct {
		TempC     float64 `json:"temp_c"`
		TempF     float64 `json:"temp_f"`
		Condition struct {
			Text string `json:"text"`
		} `json:"condition"`
		Humidity   int     `json:"humidity"`
		FeelsLikeC float64 `json:"feelslike_c"`
		FeelsLikeF float64 `json:"feelslike_f"`
		WindMph    float64 `json:"wind_mph"`
		WindKph    float64 `json:"wind_kph"`
	} `json:"current"`
}

func (ws *SimpleWeatherService) GetWeather(ctx context.Context, location string) (string, error) {
	if ws.apiKey == "" {
		return "Weather service not configured. Please set WEATHER_API_KEY.", nil
	}

	// Build the API request URL for WeatherAPI.com
	url := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no", ws.apiKey, location)
	
	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Execute the request
	resp, err := ws.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get weather data: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("location '%s' not found", location)
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return "", errors.New("invalid API key for weather service")
		}
		if resp.StatusCode == http.StatusBadRequest {
			return "", errors.New("invalid location format or missing parameters")
		}
		return "", fmt.Errorf("weather API returned status: %s", resp.Status)
	}

	// Parse the response
	var weatherData WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&weatherData); err != nil {
		return "", fmt.Errorf("failed to parse weather data: %w", err)
	}

	// Format the weather information
	locationName := weatherData.Location.Name
	if weatherData.Location.Region != "" {
		locationName += ", " + weatherData.Location.Region
	}
	if weatherData.Location.Country != "" {
		locationName += ", " + weatherData.Location.Country
	}

	temperature := int(weatherData.Current.TempF)
	weatherDescription := weatherData.Current.Condition.Text
	humidity := weatherData.Current.Humidity
	feelsLike := int(weatherData.Current.FeelsLikeF)

	return fmt.Sprintf("Weather in %s: %d°F (%s), Feels like: %d°F, Humidity: %d%%, Wind: %.1f mph", 
		locationName, temperature, weatherDescription, feelsLike, humidity, weatherData.Current.WindMph), nil
}

type WeatherTool struct {
	weatherService *SimpleWeatherService
}

func NewWeatherTool(apiKey string) *WeatherTool {
	return &WeatherTool{
		weatherService: NewSimpleWeatherService(apiKey),
	}
}

func (w *WeatherTool) Name() string {
	return "get_weather"
}

func (w *WeatherTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        w.Name(),
		Description: openai.String("Get current weather conditions at the given location"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]string{
					"type":        "string",
					"description": "City name, ZIP code, or latitude,longitude coordinates",
				},
			},
			"required": []string{"location"},
		},
	})
}

func (w *WeatherTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var payload struct {
		Location string `json:"location"`
	}

	if err := json.Unmarshal(args, &payload); err != nil {
		return "", err
	}

	if payload.Location == "" {
		return "", errors.New("location is required for weather information")
	}

	return w.weatherService.GetWeather(ctx, payload.Location)
}