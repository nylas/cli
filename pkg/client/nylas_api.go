package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type NylasAPI struct {
	BaseURL    string
	AppAPIKey  string
	HttpClient *http.Client
}

func CreateNylasAPIClient(BaseURL string) *NylasAPI {
	return &NylasAPI{
		BaseURL:    BaseURL,
		HttpClient: &http.Client{},
	}
}

type Data struct {
	RequestID string                 `json:"request_id"`
	Data      map[string]interface{} `json:"data"`
}

type IPAddresses struct {
	IPAddresses []string `json:"ip_addresses,omitempty" bson:",omitempty"`
	UpdatedAt   int      `json:"updated_at,omitempty" bson:",omitempty"`
}

type APIError struct {
	Success bool `json:"success"`
	Error   struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (nylasAPI *NylasAPI) RawRequest(method string, path string, body io.Reader, headers map[string]string) ([]byte, http.Header, error) {
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(nylasAPI.BaseURL, "/"), strings.TrimPrefix(path, "/"))
	req, err := http.NewRequest(strings.ToUpper(method), url, body)
	if err != nil {
		return nil, nil, err
	}

	for key, value := range headers {
		req.Header.Add(key, value) // How we set the Bearer token
	}

	resp, err := nylasAPI.HttpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		apiError := APIError{}
		if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
			jsonError := json.Unmarshal(bodyBytes, &apiError)
			if jsonError != nil {
				return nil, nil, jsonError
			}
		} else {
			apiError.Error.Type = "unknown"
			apiError.Error.Message = string(bodyBytes)
		}

		errorType := apiError.Error.Type
		if errorType == "" {
			errorType = "api_error"
		}

		return nil, nil, fmt.Errorf("[%d][%s] %s", resp.StatusCode, errorType, apiError.Error.Message)
	}

	return bodyBytes, resp.Header, nil
}

func (nylasAPI *NylasAPI) Request(response interface{}, method string, path string, requestBody io.Reader, apiKey string) error {
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(nylasAPI.BaseURL, "/"), strings.TrimPrefix(path, "/"))
	req, err := http.NewRequest(strings.ToUpper(method), url, requestBody)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	if apiKey != "" {
		req.Header.Add("Authorization", apiKey) // How we set the Bearer token
	}
	if err != nil {
		return err
	}
	resp, err := nylasAPI.HttpClient.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		apiError := APIError{}
		if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
			jsonError := json.Unmarshal(body, &apiError)
			if jsonError != nil {
				return jsonError
			}
		} else {
			apiError.Error.Type = "unknown"
			apiError.Error.Message = string(body)
		}
		errorType := apiError.Error.Type
		if errorType == "" {
			errorType = "api_error"
		}

		return fmt.Errorf("[%d][%s] %s", resp.StatusCode, errorType, apiError.Error.Message)
	}

	var data Data
	if jsonError := json.Unmarshal(body, &data); jsonError != nil {
		return jsonError
	}
	body, jsonError := json.Marshal(data.Data)
	if jsonError != nil {
		return jsonError
	}
	if jsonError := json.Unmarshal(body, &response); jsonError != nil {
		return jsonError
	}

	return err
}
