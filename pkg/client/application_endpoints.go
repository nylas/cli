package client

import (
	"bytes"
	"encoding/json"
	"net/http"
)

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

func (nylasAPI *NylasAPI) GetApplication(apiKey string) (Application, error) {
	var application Application

	err := nylasAPI.Request(&application, http.MethodGet,
		"/v3/applications", nil, "Bearer "+apiKey)
	return application, err
}

func (nylasAPI *NylasAPI) UpdateApplication(application Application, apiKey string) (Application, error) {
	var updatedApplication Application

	jsonString, jsonError := json.Marshal(application)
	if jsonError != nil {
		return updatedApplication, jsonError
	}

	err := nylasAPI.Request(&updatedApplication, http.MethodPatch,
		"/v3/applications", bytes.NewBuffer(jsonString), "Bearer "+apiKey)
	return updatedApplication, err
}

func (nylasAPI *NylasAPI) ListGrants(apiKey string) ([]Grant, error) {
	var grants []Grant
	err := nylasAPI.Request(&grants, http.MethodGet,
		"/v3/grants", nil, "Bearer "+apiKey)
	return grants, err
}

func (nylasAPI *NylasAPI) GetGrant(apiKey, grantID string) (Grant, error) {
	var grant Grant
	err := nylasAPI.Request(&grant, http.MethodGet,
		"/v3/grants/"+grantID, nil, "Bearer "+apiKey)
	return grant, err
}

func (nylasAPI *NylasAPI) DeleteGrant(apiKey, grantID string) error {
	return nylasAPI.Request(nil, http.MethodDelete,
		"/v3/grants/"+grantID, nil, "Bearer "+apiKey)
}
