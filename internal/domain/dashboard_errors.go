package domain

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

// DashboardAPIError preserves structured dashboard API failures while allowing
// callers to detect specific auth states with errors.Is.
type DashboardAPIError struct {
	StatusCode int
	Code       string
	ServerMsg  string
	sentinel   error
}

// NewDashboardAPIError creates a dashboard API error and classifies it when a
// domain-level sentinel applies.
func NewDashboardAPIError(statusCode int, code, message string) *DashboardAPIError {
	serverMsg := message
	if code != "" {
		if message != "" {
			serverMsg = code + ": " + message
		} else {
			serverMsg = code
		}
	}

	return &DashboardAPIError{
		StatusCode: statusCode,
		Code:       code,
		ServerMsg:  serverMsg,
		sentinel:   classifyDashboardAPIError(statusCode, code),
	}
}

func (e *DashboardAPIError) Error() string {
	if e == nil {
		return "dashboard API error"
	}
	if e.ServerMsg != "" {
		return fmt.Sprintf("dashboard API error (HTTP %d): %s", e.StatusCode, e.ServerMsg)
	}
	return fmt.Sprintf("dashboard API error (HTTP %d)", e.StatusCode)
}

func (e *DashboardAPIError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.sentinel
}

func classifyDashboardAPIError(statusCode int, code string) error {
	if statusCode == 401 && code == "INVALID_SESSION" {
		return ErrDashboardSessionExpired
	}
	return nil
}

// DashboardPartialResultError indicates that a dashboard operation returned a
// usable partial result set while one or more backends failed.
type DashboardPartialResultError struct {
	Operation string
	Failures  map[string]error
}

func (e *DashboardPartialResultError) Error() string {
	if e == nil {
		return "dashboard operation returned partial results"
	}

	operation := e.Operation
	if operation == "" {
		operation = "dashboard operation"
	}

	parts := make([]string, 0, len(e.Failures))
	regions := make([]string, 0, len(e.Failures))
	for region := range e.Failures {
		regions = append(regions, region)
	}
	sort.Strings(regions)
	for _, region := range regions {
		parts = append(parts, fmt.Sprintf("%s: %v", region, e.Failures[region]))
	}
	return fmt.Sprintf("%s returned partial results (%s)", operation, strings.Join(parts, "; "))
}

func (e *DashboardPartialResultError) Unwrap() error {
	if e == nil || len(e.Failures) == 0 {
		return nil
	}

	errs := make([]error, 0, len(e.Failures))
	for _, err := range e.Failures {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
