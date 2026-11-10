package tenant

import (
	"strings"
	"testing"
)

func TestTenantValidate(t *testing.T) {
	tests := []struct {
		name       string
		tenant     Tenant
		wantErrors []string
	}{
		{
			name: "valid tenant",
			tenant: Tenant{
				Name:               "Demo Business",
				WABAID:             "2161887421343402",
				PhoneNumberID:      "1257434784125832",
				DisplayPhoneNumber: "15552050459",
				AccessToken:        "secret-token",
			},
		},
		{
			name:       "required fields",
			tenant:     Tenant{},
			wantErrors: []string{"name", "waba_id", "phone_number_id", "display_phone_number", "access_token"},
		},
		{
			name: "maximum lengths",
			tenant: Tenant{
				Name:               strings.Repeat("a", 151),
				WABAID:             strings.Repeat("1", 101),
				PhoneNumberID:      strings.Repeat("2", 101),
				DisplayPhoneNumber: strings.Repeat("3", 31),
				AccessToken:        "secret-token",
			},
			wantErrors: []string{"name", "waba_id", "phone_number_id", "display_phone_number"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.tenant.Validate()
			if len(errors) != len(tt.wantErrors) {
				t.Fatalf("expected %d errors, got %d: %v", len(tt.wantErrors), len(errors), errors)
			}
			for _, field := range tt.wantErrors {
				if errors[field] == "" {
					t.Errorf("expected a validation error for %q", field)
				}
			}
		})
	}
}
