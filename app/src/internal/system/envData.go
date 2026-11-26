package system

import (
	"fmt"
	"maps"
	"os"
)

const (
	DISCORD_BACKUP_DIR    = "DISCORD_BACKUP_DIR"
	MATRIX_SERVER_ADDRESS = "MATRIX_SERVER_ADDRESS"
	MATRIX_TOKEN          = "MATRIX_TOKEN"
	MATRIX_USERNAME       = "MATRIX_USERNAME"
	MATRIX_PASSWORD       = "MATRIX_PASSWORD"
)

func GetEnv() (map[string]string, error) {
	envs, err := checkRequiredEnvs()
	if err != nil {
		return nil, err
	}

	optionalEnvs, err := checkOptionalEnvs()
	if err != nil {
		return nil, err
	}

	maps.Copy(envs, optionalEnvs)

	return envs, nil
}

func checkRequiredEnvs() (map[string]string, error) {
	var requiredEnvs = [2]string{DISCORD_BACKUP_DIR, MATRIX_SERVER_ADDRESS}
	var filledEnvs = make(map[string]string)
	for _, env := range requiredEnvs {

		envValue := os.Getenv(env)

		if envValue == "" {
			return nil, fmt.Errorf("missing env %s. Your docker-compose.yaml file must contains the following environment variable %s", env, requiredEnvs)
		}

		filledEnvs[env] = envValue
	}

	return filledEnvs, nil

}

func checkOptionalEnvs() (map[string]string, error) {

	var optionalEnvs = [3]string{MATRIX_TOKEN, MATRIX_USERNAME, MATRIX_PASSWORD}
	var filledEnvs = make(map[string]string)

	for _, env := range optionalEnvs {
		filledEnvs[env] = os.Getenv(env)
	}

	if filledEnvs[MATRIX_TOKEN] == "" {
		if filledEnvs[MATRIX_USERNAME] == "" || filledEnvs[MATRIX_PASSWORD] == "" {
			return nil, fmt.Errorf("you must provide either a valid matrix token or your credentials to login to your server")

		}
	}

	return filledEnvs, nil
}
