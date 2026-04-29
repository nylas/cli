//go:build integration
// +build integration

package integration

import "strings"

// extractEventID extracts event ID from CLI output
func extractEventID(output string) string {
	// Look for event ID patterns in output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Look for "ID: <id>" or "Event ID: <id>"
		if strings.Contains(line, "Event ID:") || strings.Contains(line, "ID:") {
			parts := strings.Fields(line)
			for i, part := range parts {
				if (part == "ID:" || part == "Event" && i+1 < len(parts) && parts[i+1] == "ID:") && i+1 < len(parts) {
					// Next field should be the ID
					nextIdx := i + 1
					if part == "Event" {
						nextIdx = i + 2
					}
					if nextIdx < len(parts) {
						return parts[nextIdx]
					}
				}
			}
		}
		// Also try to match event_* or cal_event_* patterns
		if strings.Contains(line, "event_") || strings.Contains(line, "cal_event_") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "event_") || strings.HasPrefix(part, "cal_event_") {
					// Clean up any trailing punctuation
					id := strings.TrimRight(part, ".,;:\"'")
					return id
				}
			}
		}
	}
	return ""
}

// extractEventIDFromList extracts event ID from list output by finding title
func extractEventIDFromList(output, title string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, title) {
			// Look for ID in the same line or nearby lines
			if strings.Contains(line, "ID:") {
				parts := strings.Split(line, "ID:")
				if len(parts) > 1 {
					idPart := strings.TrimSpace(parts[1])
					fields := strings.Fields(idPart)
					if len(fields) > 0 {
						return fields[0]
					}
				}
			}
			// Check previous lines for ID
			for j := i - 1; j >= 0 && j >= i-3; j-- {
				if strings.Contains(lines[j], "ID:") {
					parts := strings.Split(lines[j], "ID:")
					if len(parts) > 1 {
						idPart := strings.TrimSpace(parts[1])
						fields := strings.Fields(idPart)
						if len(fields) > 0 {
							return fields[0]
						}
					}
				}
			}
		}
	}
	return ""
}
