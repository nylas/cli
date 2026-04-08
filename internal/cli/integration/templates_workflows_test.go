//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nylas/cli/internal/domain"
)

func TestCLI_TemplateHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("template", "--help")
	if err != nil {
		t.Fatalf("template --help failed: %v\nstderr: %s", err, stderr)
	}

	for _, expected := range []string{"list", "show", "create", "update", "delete", "render", "render-html"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected help output to contain %q, got: %s", expected, stdout)
		}
	}
}

func TestCLI_WorkflowHelp(t *testing.T) {
	if testBinary == "" {
		t.Skip("CLI binary not found")
	}

	stdout, stderr, err := runCLI("workflow", "--help")
	if err != nil {
		t.Fatalf("workflow --help failed: %v\nstderr: %s", err, stderr)
	}

	for _, expected := range []string{"list", "show", "create", "update", "delete"} {
		if !strings.Contains(stdout, expected) {
			t.Fatalf("expected help output to contain %q, got: %s", expected, stdout)
		}
	}
}

func TestCLI_TemplateCRUD(t *testing.T) {
	skipIfMissingCreds(t)

	createStdout, createStderr, createErr := runCLIWithRateLimit(t,
		"template", "create",
		"--name", "Integration Template",
		"--subject", "Hello {{user.name}}",
		"--body", "<p>Hello {{user.name}}</p>",
		"--engine", "mustache",
		"--json",
	)
	if createErr != nil {
		t.Fatalf("template create failed: %v\nstderr: %s", createErr, createStderr)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(createStdout)), &created); err != nil {
		t.Fatalf("failed to parse template create output: %v\noutput: %s", err, createStdout)
	}
	if created.ID == "" {
		t.Fatalf("template create did not return an id: %s", createStdout)
	}

	t.Cleanup(func() {
		if created.ID == "" {
			return
		}
		_, _, _ = runCLI("template", "delete", created.ID, "--yes")
	})

	showStdout, showStderr, showErr := runCLIWithRateLimit(t, "template", "show", created.ID, "--json")
	if showErr != nil {
		t.Fatalf("template show failed: %v\nstderr: %s", showErr, showStderr)
	}
	if !strings.Contains(showStdout, created.ID) {
		t.Fatalf("expected template show output to contain %s, got: %s", created.ID, showStdout)
	}

	updateStdout, updateStderr, updateErr := runCLIWithRateLimit(t,
		"template", "update", created.ID,
		"--name", "Integration Template Updated",
		"--json",
	)
	if updateErr != nil {
		t.Fatalf("template update failed: %v\nstderr: %s", updateErr, updateStderr)
	}
	if !strings.Contains(updateStdout, "Integration Template Updated") {
		t.Fatalf("expected updated template name in output, got: %s", updateStdout)
	}

	renderStdout, renderStderr, renderErr := runCLIWithRateLimit(t,
		"template", "render", created.ID,
		"--data", `{"user":{"name":"Integration"}}`,
		"--json",
	)
	if renderErr != nil {
		t.Fatalf("template render failed: %v\nstderr: %s", renderErr, renderStderr)
	}
	if !strings.Contains(renderStdout, "Integration") {
		t.Fatalf("expected rendered template to contain substituted value, got: %s", renderStdout)
	}

	deleteStdout, deleteStderr, deleteErr := runCLIWithRateLimit(t, "template", "delete", created.ID, "--yes")
	if deleteErr != nil {
		t.Fatalf("template delete failed: %v\nstderr: %s", deleteErr, deleteStderr)
	}
	if !strings.Contains(deleteStdout, "Template deleted") {
		t.Fatalf("expected delete confirmation, got: %s", deleteStdout)
	}
	created.ID = ""
}

func TestCLI_TemplateListAndGrantScopedFileFlows(t *testing.T) {
	skipIfMissingCreds(t)
	grantIdentifier := getGrantEmail(t)
	envOverrides := newSeededGrantStoreEnv(t, domain.GrantInfo{ID: testGrantID, Email: grantIdentifier})

	tempDir := t.TempDir()
	bodyPath := filepath.Join(tempDir, "template-body.html")
	if err := os.WriteFile(bodyPath, []byte("<p>Hello {{user.name}}</p>\n"), 0o600); err != nil {
		t.Fatalf("failed to write template body file: %v", err)
	}

	createStdout, createStderr, createErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"template", "create",
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--name", "Grant File Integration Template",
		"--subject", "Grant Hello {{user.name}}",
		"--body-file", bodyPath,
		"--engine", "mustache",
		"--json",
	)
	if createErr != nil {
		t.Fatalf("grant-scoped template create failed: %v\nstderr: %s", createErr, createStderr)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(createStdout)), &created); err != nil {
		t.Fatalf("failed to parse template create output: %v\noutput: %s", err, createStdout)
	}
	if created.ID == "" {
		t.Fatalf("template create did not return an id: %s", createStdout)
	}

	t.Cleanup(func() {
		if created.ID != "" {
			_, _, _ = runCLIWithOverrides(2*time.Minute, envOverrides,
				"template", "delete", created.ID,
				"--scope", "grant",
				"--grant-id", grantIdentifier,
				"--yes",
			)
		}
	})

	listStdout, listStderr, listErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"template", "list",
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--limit", "100",
		"--json",
	)
	if listErr != nil {
		t.Fatalf("grant-scoped template list failed: %v\nstderr: %s", listErr, listStderr)
	}
	if !strings.Contains(listStdout, created.ID) {
		t.Fatalf("expected template list output to contain %s, got: %s", created.ID, listStdout)
	}

	dataPath := filepath.Join(tempDir, "template-data.json")
	if err := os.WriteFile(dataPath, []byte(`{"user":{"name":"Grant Integration"}}`), 0o600); err != nil {
		t.Fatalf("failed to write template data file: %v", err)
	}

	renderStdout, renderStderr, renderErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"template", "render", created.ID,
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--data-file", dataPath,
		"--json",
	)
	if renderErr != nil {
		t.Fatalf("grant-scoped template render failed: %v\nstderr: %s", renderErr, renderStderr)
	}
	if !strings.Contains(renderStdout, "Grant Hello Grant Integration") {
		t.Fatalf("expected rendered subject in output, got: %s", renderStdout)
	}
	var renderResult struct {
		Body    string `json:"body"`
		Subject string `json:"subject"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(renderStdout)), &renderResult); err != nil {
		t.Fatalf("failed to parse render output: %v\noutput: %s", err, renderStdout)
	}
	if renderResult.Body != "<p>Hello Grant Integration</p>" {
		t.Fatalf("rendered body = %q, want %q", renderResult.Body, "<p>Hello Grant Integration</p>")
	}

	renderHTMLPath := filepath.Join(tempDir, "render-html-body.html")
	if err := os.WriteFile(renderHTMLPath, []byte("<div>{{user.name}} via render-html</div>\n"), 0o600); err != nil {
		t.Fatalf("failed to write render-html body file: %v", err)
	}

	renderHTMLStdout, renderHTMLStderr, renderHTMLErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"template", "render-html",
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--body-file", renderHTMLPath,
		"--engine", "mustache",
		"--data-file", dataPath,
		"--strict=false",
		"--json",
	)
	if renderHTMLErr != nil {
		t.Fatalf("grant-scoped template render-html failed: %v\nstderr: %s", renderHTMLErr, renderHTMLStderr)
	}
	if !strings.Contains(renderHTMLStdout, "Grant Integration via render-html") {
		t.Fatalf("expected rendered HTML output, got: %s", renderHTMLStdout)
	}

	updateBodyPath := filepath.Join(tempDir, "template-body-updated.html")
	if err := os.WriteFile(updateBodyPath, []byte("<p>Updated {{user.name}}</p>\n"), 0o600); err != nil {
		t.Fatalf("failed to write updated template body file: %v", err)
	}

	updateStdout, updateStderr, updateErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"template", "update", created.ID,
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--name", "Grant File Integration Template Updated",
		"--body-file", updateBodyPath,
		"--json",
	)
	if updateErr != nil {
		t.Fatalf("grant-scoped template update failed: %v\nstderr: %s", updateErr, updateStderr)
	}
	if !strings.Contains(updateStdout, "Grant File Integration Template Updated") {
		t.Fatalf("expected updated template name in output, got: %s", updateStdout)
	}

	showStdout, showStderr, showErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"template", "show", created.ID,
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--json",
	)
	if showErr != nil {
		t.Fatalf("grant-scoped template show failed: %v\nstderr: %s", showErr, showStderr)
	}
	if !strings.Contains(showStdout, "Grant File Integration Template Updated") {
		t.Fatalf("expected updated template in show output, got: %s", showStdout)
	}

	deleteStdout, deleteStderr, deleteErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"template", "delete", created.ID,
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--yes",
	)
	if deleteErr != nil {
		t.Fatalf("grant-scoped template delete failed: %v\nstderr: %s", deleteErr, deleteStderr)
	}
	if !strings.Contains(deleteStdout, "Template deleted") {
		t.Fatalf("expected delete confirmation, got: %s", deleteStdout)
	}
	created.ID = ""
}

func TestCLI_WorkflowCRUD(t *testing.T) {
	skipIfMissingCreds(t)

	templateStdout, templateStderr, templateErr := runCLIWithRateLimit(t,
		"template", "create",
		"--name", "Workflow Integration Template",
		"--subject", "Booking {{user.name}}",
		"--body", "<p>Booking {{user.name}}</p>",
		"--engine", "mustache",
		"--json",
	)
	if templateErr != nil {
		t.Fatalf("template create for workflow failed: %v\nstderr: %s", templateErr, templateStderr)
	}

	var template struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(templateStdout)), &template); err != nil {
		t.Fatalf("failed to parse template create output: %v\noutput: %s", err, templateStdout)
	}
	if template.ID == "" {
		t.Fatalf("expected template id, got: %s", templateStdout)
	}

	t.Cleanup(func() {
		if template.ID != "" {
			_, _, _ = runCLI("template", "delete", template.ID, "--yes")
		}
	})

	createStdout, createStderr, createErr := runCLIWithRateLimit(t,
		"workflow", "create",
		"--name", "Integration Workflow",
		"--template-id", template.ID,
		"--trigger-event", "booking.created",
		"--delay", "1",
		"--enabled",
		"--json",
	)
	if createErr != nil {
		t.Fatalf("workflow create failed: %v\nstderr: %s", createErr, createStderr)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(createStdout)), &created); err != nil {
		t.Fatalf("failed to parse workflow create output: %v\noutput: %s", err, createStdout)
	}
	if created.ID == "" {
		t.Fatalf("workflow create did not return an id: %s", createStdout)
	}

	t.Cleanup(func() {
		if created.ID != "" {
			_, _, _ = runCLI("workflow", "delete", created.ID, "--yes")
		}
	})

	showStdout, showStderr, showErr := runCLIWithRateLimit(t, "workflow", "show", created.ID, "--json")
	if showErr != nil {
		t.Fatalf("workflow show failed: %v\nstderr: %s", showErr, showStderr)
	}
	if !strings.Contains(showStdout, created.ID) {
		t.Fatalf("expected workflow show output to contain %s, got: %s", created.ID, showStdout)
	}

	updateStdout, updateStderr, updateErr := runCLIWithRateLimit(t,
		"workflow", "update", created.ID,
		"--name", "Integration Workflow Updated",
		"--disabled",
		"--json",
	)
	if updateErr != nil {
		t.Fatalf("workflow update failed: %v\nstderr: %s", updateErr, updateStderr)
	}
	if !strings.Contains(updateStdout, "Integration Workflow Updated") {
		t.Fatalf("expected updated workflow name in output, got: %s", updateStdout)
	}

	deleteStdout, deleteStderr, deleteErr := runCLIWithRateLimit(t, "workflow", "delete", created.ID, "--yes")
	if deleteErr != nil {
		t.Fatalf("workflow delete failed: %v\nstderr: %s", deleteErr, deleteStderr)
	}
	if !strings.Contains(deleteStdout, "Workflow deleted") {
		t.Fatalf("expected delete confirmation, got: %s", deleteStdout)
	}
	created.ID = ""
}

func TestCLI_WorkflowGrantScopeFileAndListFlows(t *testing.T) {
	skipIfMissingCreds(t)
	grantIdentifier := getGrantEmail(t)
	envOverrides := newSeededGrantStoreEnv(t, domain.GrantInfo{ID: testGrantID, Email: grantIdentifier})

	templateStdout, templateStderr, templateErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"template", "create",
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--name", "Grant Workflow Template",
		"--subject", "Grant Workflow {{user.name}}",
		"--body", "<p>Grant Workflow {{user.name}}</p>",
		"--engine", "mustache",
		"--json",
	)
	if templateErr != nil {
		t.Fatalf("template create for workflow file test failed: %v\nstderr: %s", templateErr, templateStderr)
	}

	var template struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(templateStdout)), &template); err != nil {
		t.Fatalf("failed to parse template create output: %v\noutput: %s", err, templateStdout)
	}
	if template.ID == "" {
		t.Fatalf("expected template id, got: %s", templateStdout)
	}

	t.Cleanup(func() {
		if template.ID != "" {
			_, _, _ = runCLIWithOverrides(2*time.Minute, envOverrides,
				"template", "delete", template.ID,
				"--scope", "grant",
				"--grant-id", grantIdentifier,
				"--yes",
			)
		}
	})

	tempDir := t.TempDir()
	createPath := filepath.Join(tempDir, "workflow-create.json")
	createPayload := []byte(`{
  "name": "Grant Workflow File Create",
  "template_id": "` + template.ID + `",
  "trigger_event": "booking.created",
  "delay": 2,
  "is_enabled": true
}`)
	if err := os.WriteFile(createPath, createPayload, 0o600); err != nil {
		t.Fatalf("failed to write workflow create file: %v", err)
	}

	createStdout, createStderr, createErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"workflow", "create",
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--file", createPath,
		"--json",
	)
	if createErr != nil {
		t.Fatalf("grant-scoped workflow create failed: %v\nstderr: %s", createErr, createStderr)
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(createStdout)), &created); err != nil {
		t.Fatalf("failed to parse workflow create output: %v\noutput: %s", err, createStdout)
	}
	if created.ID == "" {
		t.Fatalf("workflow create did not return an id: %s", createStdout)
	}

	t.Cleanup(func() {
		if created.ID != "" {
			_, _, _ = runCLIWithOverrides(2*time.Minute, envOverrides,
				"workflow", "delete", created.ID,
				"--scope", "grant",
				"--grant-id", grantIdentifier,
				"--yes",
			)
		}
	})

	listStdout, listStderr, listErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"workflow", "list",
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--limit", "100",
		"--json",
	)
	if listErr != nil {
		t.Fatalf("grant-scoped workflow list failed: %v\nstderr: %s", listErr, listStderr)
	}
	if !strings.Contains(listStdout, created.ID) {
		t.Fatalf("expected workflow list output to contain %s, got: %s", created.ID, listStdout)
	}

	updatePath := filepath.Join(tempDir, "workflow-update.json")
	updatePayload := []byte(`{
  "name": "Grant Workflow File Updated",
  "delay": 5,
  "is_enabled": false
}`)
	if err := os.WriteFile(updatePath, updatePayload, 0o600); err != nil {
		t.Fatalf("failed to write workflow update file: %v", err)
	}

	updateStdout, updateStderr, updateErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"workflow", "update", created.ID,
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--file", updatePath,
		"--json",
	)
	if updateErr != nil {
		t.Fatalf("grant-scoped workflow update failed: %v\nstderr: %s", updateErr, updateStderr)
	}
	if !strings.Contains(updateStdout, "Grant Workflow File Updated") {
		t.Fatalf("expected updated workflow name in output, got: %s", updateStdout)
	}

	showStdout, showStderr, showErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"workflow", "show", created.ID,
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--json",
	)
	if showErr != nil {
		t.Fatalf("grant-scoped workflow show failed: %v\nstderr: %s", showErr, showStderr)
	}
	if !strings.Contains(showStdout, "Grant Workflow File Updated") {
		t.Fatalf("expected updated workflow in show output, got: %s", showStdout)
	}
	if !strings.Contains(showStdout, `"is_enabled": false`) {
		t.Fatalf("expected workflow show output to reflect disabled state, got: %s", showStdout)
	}

	deleteStdout, deleteStderr, deleteErr := runCLIWithOverridesAndRateLimit(t, 2*time.Minute, envOverrides,
		"workflow", "delete", created.ID,
		"--scope", "grant",
		"--grant-id", grantIdentifier,
		"--yes",
	)
	if deleteErr != nil {
		t.Fatalf("grant-scoped workflow delete failed: %v\nstderr: %s", deleteErr, deleteStderr)
	}
	if !strings.Contains(deleteStdout, "Workflow deleted") {
		t.Fatalf("expected delete confirmation, got: %s", deleteStdout)
	}
	created.ID = ""
}
