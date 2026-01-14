package email

import (
	"context"
	"testing"

	"github.com/nylas/cli/internal/domain"
)

func TestNewService(t *testing.T) {
	service := NewService()
	if service == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestBuildTemplate_MissingName(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	req := &domain.TemplateRequest{
		Subject: "Test",
	}

	_, err := service.BuildTemplate(ctx, req)
	if err == nil {
		t.Error("expected error for missing template name")
	}
	if err != nil && err.Error() != "template name is required" {
		t.Errorf("expected 'template name is required' error, got: %v", err)
	}
}

func TestBuildTemplate_MissingSubject(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	req := &domain.TemplateRequest{
		Name: "test-template",
	}

	_, err := service.BuildTemplate(ctx, req)
	if err == nil {
		t.Error("expected error for missing subject")
	}
	if err != nil && err.Error() != "subject is required" {
		t.Errorf("expected 'subject is required' error, got: %v", err)
	}
}

func TestBuildTemplate_Valid(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	req := &domain.TemplateRequest{
		Name:      "welcome",
		Subject:   "Welcome to our service!",
		HTMLBody:  "<p>Hello {{name}}</p>",
		TextBody:  "Hello {{name}}",
		Variables: []string{"name"},
	}

	template, err := service.BuildTemplate(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if template.Name != "welcome" {
		t.Errorf("expected name=welcome, got %s", template.Name)
	}
	if template.Subject != "Welcome to our service!" {
		t.Errorf("expected subject='Welcome to our service!', got %s", template.Subject)
	}
	if len(template.Variables) != 1 {
		t.Errorf("expected 1 variable, got %d", len(template.Variables))
	}
}

func TestPreviewTemplate(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	template := &domain.EmailTemplate{
		HTMLBody: "<p>Hello {{name}}, your score is {{score}}</p>",
	}

	data := map[string]any{
		"name":  "Alice",
		"score": 95,
	}

	result, err := service.PreviewTemplate(ctx, template, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "<p>Hello Alice, your score is 95</p>"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
