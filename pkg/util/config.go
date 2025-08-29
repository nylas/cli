package util

type RegionConfiguration struct {
	NylasAPIURL       string
	DashboardAPIURL   string
	StreamEndpointURL string
	CallbackDomain    string
	WebsocketDomain   string
	TelemetryAPIURL   string
}

var RegionConfig = map[string]RegionConfiguration{
	"us": {
		NylasAPIURL:       "https://api.us.nylas.com",
		DashboardAPIURL:   "https://dashboard-api.nylas.com",
		StreamEndpointURL: "http://localhost:8080/stream",
		CallbackDomain:    "cb.nylas.com",
		TelemetryAPIURL:   "https://cli.nylas.com",
	},
	"eu": {
		NylasAPIURL:       "https://api.eu.nylas.com",
		DashboardAPIURL:   "https://dashboard-api.nylas.com",
		StreamEndpointURL: "http://localhost:8080/stream",
		CallbackDomain:    "cb.nylas.com",
		TelemetryAPIURL:   "https://cli.nylas.com",
	},
}
