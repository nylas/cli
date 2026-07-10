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
		wantErr    bool
	}{
		// --grant must be honored when no positional grant is given; two
		// DIFFERENT grants at once is ambiguous and must be rejected rather
		// than silently picking one (wrong pick = mutating another account).
		{name: "flag used when no positional", positional: nil, grantFlag: "grant-flag", want: []string{"grant-flag"}},
		{name: "no positional, no flag falls back to default", positional: nil, grantFlag: "", want: nil},
		{name: "positional used with no flag", positional: []string{"grant-pos"}, grantFlag: "", want: []string{"grant-pos"}},
		{name: "same grant in both is accepted", positional: []string{"grant-1"}, grantFlag: "grant-1", want: []string{"grant-1"}},
		{name: "conflicting grants are rejected", positional: []string{"grant-pos"}, grantFlag: "grant-flag", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := recurringGrantArgs(tt.positional, tt.grantFlag)
			if (err != nil) != tt.wantErr {
				t.Fatalf("recurringGrantArgs(%v, %q) error = %v, wantErr %v", tt.positional, tt.grantFlag, err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("recurringGrantArgs(%v, %q) = %v, want %v", tt.positional, tt.grantFlag, got, tt.want)
			}
		})
	}
}
