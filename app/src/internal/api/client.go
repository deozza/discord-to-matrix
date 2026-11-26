package client

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

type ApiClient struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func GetApiClient(baseUrl string) *ApiClient {
	return &ApiClient{
		BaseURL:    "https://" + baseUrl,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *ApiClient) SendRequest(method string, url string, payload *bytes.Buffer, headers map[string]string) (int, []byte, error) {
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		return 500, nil, err
	}

	for key, value := range headers {
		req.Header.Add(key, value)
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return res.StatusCode, nil, err
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return res.StatusCode, nil, err
	}

	return res.StatusCode, body, nil
}
