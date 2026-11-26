package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"internal/api/client"
	"net/http"
)

type loginPayload struct {
	Identifier identifier `json:"identifier"`
	Password   string     `json:"password"`
	Type       string     `json:"type"`
}

type identifier struct {
	Type string `json:"type"`
	User string `json:"user"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
}

func ValidateAuth(token string, c *client.ApiClient) (bool, error) {
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	}

	statusCode, _, err := c.SendRequest(http.MethodPost, fmt.Sprintf("%s%s", c.BaseURL, "/_matrix/client/v3/account/whoami"), bytes.NewBuffer(nil), headers)

	if err != nil {
		return false, err
	}

	if statusCode != 200 {
		return false, fmt.Errorf("token is invalid")
	}

	return true, nil

}

func buildLoginPayload(username string, password string) loginPayload {
	return loginPayload{
		Identifier: identifier{
			Type: "m.id.user",
			User: username,
		},
		Password: password,
		Type:     "m.login.password",
	}
}

func GetMatrixAuthToken(username string, password string, c *client.ApiClient) (*loginResponse, error) {
	loginResponse := loginResponse{}

	payload := buildLoginPayload(username, password)
	jsonData, err := json.Marshal(payload)

	_, res, err := c.SendRequest(http.MethodPost, fmt.Sprintf("%s%s", c.BaseURL, "/_matrix/client/v3/login"), bytes.NewBuffer(jsonData), map[string]string{"Content-Type": "application/json"})

	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(res, &loginResponse); err != nil {
		return nil, err
	}

	return &loginResponse, nil
}
