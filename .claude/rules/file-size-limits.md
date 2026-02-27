# File Size Limits

Go files must be ≤500 lines (ideal) or ≤600 lines (hard max). Lines = code + comments + blanks.

**>600 lines:** Must split before completing task. Split by responsibility (types, helpers, handlers).

**500-600 lines:** Evaluate — split if multiple responsibilities, keep if cohesive.

**After splitting:** Run `make build && make test-unit && golangci-lint run --timeout=5m`

**Exceptions:** Generated code (`// Code generated`), vendored code only.
