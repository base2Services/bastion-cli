package bastion

import (
	"errors"
	"testing"
)

func TestAmiParameterForAlias(t *testing.T) {
	cases := []struct {
		name       string
		input      string
		arch       string
		want       string
		wantDetect bool
	}{
		{"windows alias is not replaced by architecture", "windows", "amazon-linux", amis["windows"], false},
		{"amazon-linux on x86_64", "amazon-linux", "amazon-linux", amis["amazon-linux"], true},
		{"amazon-linux on arm64", "amazon-linux", "amazon-linux-arm64", amis["amazon-linux-arm64"], true},
		{"amazon-linux-arm64 follows instance architecture", "amazon-linux-arm64", "amazon-linux", amis["amazon-linux"], true},
		{"unknown alias falls back to architecture", "something-else", "amazon-linux-arm64", amis["amazon-linux-arm64"], true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			detected := false
			got, err := amiParameterForAlias(tc.input, func() (string, error) {
				detected = true
				return tc.arch, nil
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			if detected != tc.wantDetect {
				t.Errorf("architecture detection called = %v, want %v", detected, tc.wantDetect)
			}
		})
	}
}

func TestAmiParameterForAliasDetectError(t *testing.T) {
	wantErr := errors.New("describe failed")
	_, err := amiParameterForAlias("amazon-linux", func() (string, error) {
		return "", wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
}
