package mcp

import (
	"encoding/json"
	"errors"
	"os"
)

const nylasServerName = "nylas"

type assistantConfigState struct {
	ConfigFileExists bool
	HasNylas         bool
	BinaryPath       string
	SchemaKey        string
}

func (a Assistant) installServerConfigKey() string {
	if a.ID == "vscode" {
		return "servers"
	}

	return "mcpServers"
}

func (a Assistant) serverConfigKeys() []string {
	keys := []string{a.installServerConfigKey()}
	if a.ID == "vscode" {
		keys = append(keys, "mcpServers")
	}

	return keys
}

func loadConfig(configPath string) (map[string]any, error) {
	// #nosec G304 -- configPath is derived from validated assistant config paths
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	config := make(map[string]any)
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config, nil
}

func loadConfigIfExists(configPath string) (map[string]any, error) {
	config, err := loadConfig(configPath)
	if err == nil {
		return config, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return make(map[string]any), nil
	}

	return nil, err
}

func writeConfig(configPath string, config map[string]any) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

func inspectAssistantConfig(a Assistant, configPath string) assistantConfigState {
	config, err := loadConfig(configPath)
	if err != nil {
		return assistantConfigState{}
	}

	state := assistantConfigState{ConfigFileExists: true}
	server, schemaKey, ok := findAssistantServer(config, a, nylasServerName)
	if !ok {
		return state
	}

	state.HasNylas = true
	state.SchemaKey = schemaKey
	state.BinaryPath, _ = server["command"].(string)

	return state
}

func findAssistantServer(config map[string]any, a Assistant, serverName string) (map[string]any, string, bool) {
	for _, key := range a.serverConfigKeys() {
		servers, ok := config[key].(map[string]any)
		if !ok {
			continue
		}

		server, ok := servers[serverName].(map[string]any)
		if ok {
			return server, key, true
		}
	}

	return nil, "", false
}

func setAssistantServer(config map[string]any, a Assistant, serverName string, serverConfig map[string]any) {
	primaryKey := a.installServerConfigKey()
	servers, ok := config[primaryKey].(map[string]any)
	if !ok {
		servers = make(map[string]any)
	}

	servers[serverName] = serverConfig
	config[primaryKey] = servers

	for _, key := range a.serverConfigKeys() {
		if key == primaryKey {
			continue
		}
		removeAssistantServerFromKey(config, key, serverName)
	}
}

func removeAssistantServer(config map[string]any, a Assistant, serverName string) bool {
	removed := false

	for _, key := range a.serverConfigKeys() {
		if removeAssistantServerFromKey(config, key, serverName) {
			removed = true
		}
	}

	return removed
}

func removeAssistantServerFromKey(config map[string]any, key, serverName string) bool {
	servers, ok := config[key].(map[string]any)
	if !ok {
		return false
	}

	if _, exists := servers[serverName]; !exists {
		return false
	}

	delete(servers, serverName)
	if len(servers) == 0 {
		delete(config, key)
		return true
	}

	config[key] = servers
	return true
}
