# Add Domain Type

Add a new domain type (model, request, response) to the nylas CLI.

**Patterns:** See `add-command/references/domain-patterns.md` for all templates (value types, request/response types, enums, query params, helpers).

## Instructions

1. Ask me for:
   - Type name (e.g., Attachment, Label)
   - Fields with types
   - Whether it needs Request/Response variants
   - Any helper methods needed

2. Create or update file in `internal/domain/`:
   - Follow templates in `add-command/references/domain-patterns.md`
   - Use JSON tags with `omitempty` for optional fields
   - Use pointer fields for optional update request fields
   - Add helper methods (DisplayName, etc.) as needed

3. Add tests to `internal/domain/domain_test.go`:
   - Table-driven tests with `t.Run()`
   - Test helper methods and validation

4. Verify:
```bash
go build ./... && go test ./internal/domain/...
```
