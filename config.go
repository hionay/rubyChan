package main

import (
	"errors"
	"os"

	_ "github.com/joho/godotenv/autoload"
)

const (
	envMatrixUsername = "MATRIX_USERNAME"
	envMatrixPassword = "MATRIX_PASSWORD"
	envMatrixServer   = "MATRIX_SERVER"
	envGoogleAPIKey   = "GOOGLE_API_KEY"
	envGoogleCX       = "GOOGLE_CX"
	envWeatherAPIKey  = "WEATHER_API_KEY"
)

const (
	defaultMatrixServer = "https://matrix-client.matrix.org"
)

type Config struct {
	MatrixServer   string
	MatrixUsername string
	MatrixPassword string
	GoogleAPIKey   string
	GoogleCX       string
	WeatherAPIKey  string
}

func NewConfig() (*Config, error) {
	matrixServer := os.Getenv(envMatrixServer)
	if matrixServer == "" {
		matrixServer = defaultMatrixServer
	}
	matrixUsername := os.Getenv(envMatrixUsername)
	if matrixUsername == "" {
		return nil, errors.New("MATRIX_USERNAME not set")
	}
	matrixPassword := os.Getenv(envMatrixPassword)
	if matrixPassword == "" {
		return nil, errors.New("MATRIX_PASSWORD not set")
	}

	googleAPIKey := os.Getenv(envGoogleAPIKey)
	googleCX := os.Getenv(envGoogleCX)
	openWeatherAPIKey := os.Getenv(envWeatherAPIKey)

	return &Config{
		MatrixServer:   matrixServer,
		MatrixUsername: matrixUsername,
		MatrixPassword: matrixPassword,
		GoogleAPIKey:   googleAPIKey,
		GoogleCX:       googleCX,
		WeatherAPIKey:  openWeatherAPIKey,
	}, nil
}
