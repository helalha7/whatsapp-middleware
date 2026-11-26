package listener

import "testing"

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"object":"whatsapp_business_account"}`)
	validSignature := signPayload(string(body), "test-secret")

	tests := []struct {
		name      string
		signature string
		want      bool
	}{
		{name: "valid signature", signature: validSignature, want: true},
		{name: "missing prefix", signature: validSignature[len("sha256="):]},
		{name: "wrong algorithm", signature: "sha1=abc"},
		{name: "malformed hex", signature: "sha256=invalid"},
		{name: "incorrect digest", signature: "sha256=0000000000000000000000000000000000000000000000000000000000000000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := verifySignature(body, []byte("test-secret"), tt.signature); got != tt.want {
				t.Errorf("expected verification=%t, got %t", tt.want, got)
			}
		})
	}
}
