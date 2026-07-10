package calendar

import (
	"reflect"
	"testing"
)

func TestRecurringGrantArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		positional []string
		grantFlag  string
		want       []string
	}{
		// --grant must be honored when no positional grant is given, and the
		// positional must win when both are set.
		{name: "flag used when no positional", positional: nil, grantFlag: "grant-flag", want: []string{"grant-flag"}},
		{name: "no positional, no flag falls back to default", positional: nil, grantFlag: "", want: nil},
		{name: "positional wins over flag", positional: []string{"grant-pos"}, grantFlag: "grant-flag", want: []string{"grant-pos"}},
		{name: "positional used with no flag", positional: []string{"grant-pos"}, grantFlag: "", want: []string{"grant-pos"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := recurringGrantArgs(tt.positional, tt.grantFlag); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("recurringGrantArgs(%v, %q) = %v, want %v", tt.positional, tt.grantFlag, got, tt.want)
			}
		})
	}
}
