package rpcserver

import "encoding/json"

func resolveGrant(grantID, defaultGrant string) (string, error) {
	if grantID != "" {
		return grantID, nil
	}
	if defaultGrant != "" {
		return defaultGrant, nil
	}
	return "", NewRPCError(InvalidParams, "grant_id required", nil)
}

func decodeParams(params json.RawMessage, v any) error {
	if len(params) == 0 {
		return nil
	}
	if err := json.Unmarshal(params, v); err != nil {
		return NewRPCError(InvalidParams, "invalid params", err.Error())
	}
	return nil
}

type deletedResult struct {
	Deleted bool `json:"deleted"`
}
