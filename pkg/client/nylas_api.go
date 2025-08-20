package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

type NylasAPI struct {
	BaseURL     string
	AppClientID string
	AppAPIKey   string
	AccessToken string
	HttpClient  *http.Client
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

type Application struct {
	ApplicationID   string `json:"application_id"`
	OrganizationID  string `json:"organization_id"`
	Region          string `json:"region"`
	Environment     string `json:"environment"`
	V2ApplicationID string `json:"v2_application_id,omitempty"`
	Branding        struct {
		Name        string `json:"name"`
		IconURL     string `json:"icon_url"`
		WebsiteURL  string `json:"website_url"`
		Description string `json:"description"`
	} `json:"branding"`
	HostedAuthentication struct {
		BackgroundImageURL string `json:"background_image_url"`
		Alignment          string `json:"alignment"`
		ColorPrimary       string `json:"color_primary"`
		ColorSecondary     string `json:"color_secondary"`
		Title              string `json:"title"`
		Subtitle           string `json:"subtitle"`
		BackgroundColor    string `json:"background_color"`
		Spacing            int    `json:"spacing"`
	} `json:"hosted_authentication"`
	CallbackURIs []struct {
		Platform string `json:"platform"`
		ID       string `json:"id"`
		URL      string `json:"url"`
	} `json:"callback_uris"`
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

type Grant struct {
	ID          string   `json:"id"`
	Provider    string   `json:"provider"`
	AccountID   string   `json:"account_id"`
	GrantStatus string   `json:"grant_status"`
	Email       string   `json:"email"`
	Scope       []string `json:"scope"`
	UserAgent   string   `json:"user_agent"`
	Settings    struct {
	} `json:"settings"`
	IP        string `json:"ip"`
	State     string `json:"state"`
	Blocked   bool   `json:"blocked"`
	CreatedAt int    `json:"created_at"`
	UpdatedAt int    `json:"updated_at"`
}

type Webhook struct {
	ID            string   `json:"id,omitempty" bson:",,omitempty"`
	ApplicationID string   `json:"application_id,omitempty" bson:",,omitempty"`
	CallbackURL   string   `json:"callback_url" bson:","`
	Provider      string   `json:"provider,omitempty" bson:",omitempty"`
	State         string   `json:"state" bson:","`
	Version       string   `json:"version,omitempty" bson:",omitempty"`
	Triggers      []string `json:"triggers" bson:","`
}

func (nylasApi *NylasAPI) RawRequest(method string, path string, body io.Reader, headers map[string]string) ([]byte, http.Header, error) {
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(nylasApi.BaseURL, "/"), strings.TrimPrefix(path, "/"))
	req, err := http.NewRequest(strings.ToUpper(method), url, body)
	if err != nil {
		return nil, nil, err
	}

	for key, value := range headers {
		req.Header.Add(key, value) // How we set the Bearer token
	}

	resp, err := nylasApi.HttpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != 200 {
		apiError := APIError{}
		if resp.Header.Get("Content-Type") == "application/json" {
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

func (nylasApi *NylasAPI) Request(response interface{}, method string, path string, requestBody io.Reader, apiKey string) error {
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(nylasApi.BaseURL, "/"), strings.TrimPrefix(path, "/"))
	req, err := http.NewRequest(strings.ToUpper(method), url, requestBody)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	if apiKey != "" {
		req.Header.Add("Authorization", apiKey) // How we set the Bearer token
	}
	if err != nil {
		return err
	}
	resp, err := nylasApi.HttpClient.Do(req)

	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		apiError := APIError{}
		if resp.Header.Get("Content-Type") == "application/json" {
			fmt.Println(string(body))
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

// APPLICATION MANAGEMENT
func (nylasApi *NylasAPI) GetApplication(apiKey string) (Application, error) {
	var application Application

	err := nylasApi.Request(&application, http.MethodGet,
		"/v3/applications", nil, "Bearer "+apiKey)
	return application, err
}

func (nylasApi *NylasAPI) UpdateApplication(application Application, apiKey string) (Application, error) {
	var updatedApplication Application

	jsonString, jsonError := json.Marshal(application)
	if jsonError != nil {
		return updatedApplication, jsonError
	}

	err := nylasApi.Request(&updatedApplication, http.MethodPatch,
		"/v3/applications", bytes.NewBuffer(jsonString), "Bearer "+apiKey)
	return updatedApplication, err
}

func (nylasApi *NylasAPI) ListGrants(apiKey string) ([]Grant, error) {
	var grants []Grant
	err := nylasApi.Request(&grants, http.MethodGet,
		"/v3/grants", nil, "Bearer "+apiKey)
	return grants, err
}

func (nylasApi *NylasAPI) GetGrant(apiKey, grantID string) (Grant, error) {
	var grant Grant
	err := nylasApi.Request(&grant, http.MethodGet,
		"/v3/grants/"+grantID, nil, "Bearer "+apiKey)
	return grant, err
}

func (nylasApi *NylasAPI) DeleteGrant(apiKey, grantID string) error {
	return nylasApi.Request(nil, http.MethodDelete,
		"/v3/grants/"+grantID, nil, "Bearer "+apiKey)
}

func (nylasApi *NylasAPI) ListWebhooks(apiKey string) ([]Webhook, error) {
	var webhooks []Webhook
	err := nylasApi.Request(&webhooks, http.MethodGet,
		"/v3/webhooks", nil, "Bearer "+apiKey)
	return webhooks, err
}

func (nylasApi *NylasAPI) CreateWebhook(apiKey string, webhook Webhook) (Webhook, error) {
	var created Webhook
	reqBody, _ := json.Marshal(webhook)
	err := nylasApi.Request(&created, http.MethodPost,
		"/v3/webhooks", bytes.NewBuffer(reqBody), "Bearer "+apiKey)
	return created, err
}

func (nylasApi *NylasAPI) DeleteWebhook(apiKey, webhookID string) error {
	return nylasApi.Request(nil, http.MethodDelete,
		"/v3/webhooks/"+webhookID, nil, "Bearer "+apiKey)
}
