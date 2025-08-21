package client

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Webhook struct {
	ID            string   `json:"id,omitempty" bson:",,omitempty"`
	ApplicationID string   `json:"application_id,omitempty" bson:",,omitempty"`
	CallbackURL   string   `json:"callback_url" bson:","`
	Provider      string   `json:"provider,omitempty" bson:",omitempty"`
	State         string   `json:"state" bson:","`
	Version       string   `json:"version,omitempty" bson:",omitempty"`
	Triggers      []string `json:"triggers" bson:","`
}

func (nylasAPI *NylasAPI) ListWebhooks(apiKey string) ([]Webhook, error) {
	var webhooks []Webhook
	err := nylasAPI.Request(&webhooks, http.MethodGet,
		"/v3/webhooks", nil, "Bearer "+apiKey)
	return webhooks, err
}

func (nylasAPI *NylasAPI) CreateWebhook(apiKey string, webhook Webhook) (Webhook, error) {
	var created Webhook
	reqBody, _ := json.Marshal(webhook)
	err := nylasAPI.Request(&created, http.MethodPost,
		"/v3/webhooks", bytes.NewBuffer(reqBody), "Bearer "+apiKey)
	return created, err
}

func (nylasAPI *NylasAPI) DeleteWebhook(apiKey, webhookID string) error {
	return nylasAPI.Request(nil, http.MethodDelete,
		"/v3/webhooks/"+webhookID, nil, "Bearer "+apiKey)
}
