package main

import (
	"fmt"
	"internal/api/client"
	"internal/matrix/auth"
	"internal/system"
	"log/slog"
	"os"
)

type kernel struct {
	logger    *slog.Logger
	env       map[string]string
	apiClient *client.ApiClient
}

func main() {
	kernel, err := boot()

	if err != nil {
		kernel.logger.Error(err.Error())
	}

	err = kernel.setAuthToken()

	if err != nil {
		kernel.logger.Error(err.Error())
	}

	// files := getFilesFromBackupDir()

	// for _, file := range files {
	// 	fileContent, structure := fileHandler(file)
	// 	structure = prepareStructureOnServer(structure)
	// 	postMessages(structure.Room, fileContent.Messages)
	// }
}

func boot() (kernel, error) {
	kernel := kernel{}
	kernel.logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	env, err := system.GetEnv()

	if err != nil {
		return kernel, err
	}

	kernel.env = env

	kernel.apiClient = client.GetApiClient(kernel.env[system.MATRIX_SERVER_ADDRESS])
	return kernel, err
}

func (kernel *kernel) setAuthToken() error {
	kernel.logger.Info("Trying to authenticate with provided token")
	err := kernel.setAuthTokenFromEnv()
	if err == nil {
		kernel.logger.Info("Authentication successful")
		kernel.apiClient.Token = kernel.env[system.MATRIX_TOKEN]
		return nil
	}

	kernel.logger.Warn(err.Error())

	kernel.logger.Info("Trying to authenticate with credentials")
	token, err := kernel.setAuthTokenFromCredentials()
	if err != nil {
		return err
	}

	kernel.logger.Info("Authentication successful")
	kernel.apiClient.Token = token
	return nil
}

func (kernel *kernel) setAuthTokenFromEnv() error {
	tokenIsValid, err := auth.ValidateAuth(kernel.env[system.MATRIX_TOKEN], kernel.apiClient)

	if !tokenIsValid || err != nil {
		return fmt.Errorf("provided token is invalid, trying to log with provided username and password")
	}

	return nil
}

func (kernel *kernel) setAuthTokenFromCredentials() (string, error) {
	if kernel.env[system.MATRIX_USERNAME] == "" || kernel.env[system.MATRIX_PASSWORD] == "" {
		return "", fmt.Errorf("MATRIX_USERNAME and/or MATRIX_PASSWORD is empty. They must be provided in order to authenticate to the Matrix server")
	}

	loginResponse, err := auth.GetMatrixAuthToken(kernel.env[system.MATRIX_USERNAME], kernel.env[system.MATRIX_PASSWORD], kernel.apiClient)

	if err != nil {
		return "", err
	}

	return loginResponse.AccessToken, nil
}
