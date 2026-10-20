package mcp

import (
	"fmt"
	"strings"
	"testing"
)

func TestIsSecretKeyCatchesTheObviousNames(t *testing.T) {
	secret := []string{
		"password", "DB_PASSWORD", "db.password",
		"APP_KEY", "JWT_SECRET", "jwt_secret",
		"stripe.secret", "MAIL_PASSWORD",
		"S3_SECRET_ACCESS_KEY", "GITHUB_CLIENT_SECRET",
		"api_token", "APIKey", "credential", "salt", "dsn",
	}
	for _, key := range secret {
		if !IsSecretKey(key) {
			t.Errorf("IsSecretKey(%q) = false, want true", key)
		}
	}

	open := []string{"host", "port", "database", "driver", "APP_ENV", "lifetime"}
	for _, key := range open {
		if IsSecretKey(key) {
			t.Errorf("IsSecretKey(%q) = true, want false", key)
		}
	}
}

// The case a key-based rule alone gets wrong, and the reason this file is not
// twenty lines: DATABASE_URL is the standard name for a connection string, its
// name contains no secret-looking fragment, and its value contains the
// password.
func TestRedactValueStripsCredentialsFromAConnectionURL(t *testing.T) {
	const dsn = "postgres://appuser:s3cr3t-pw@db.example.com:5432/appdb?sslmode=require"

	got, ok := RedactValue("DATABASE_URL", dsn).(string)
	if !ok {
		t.Fatalf("RedactValue returned %T, want string", RedactValue("DATABASE_URL", dsn))
	}
	if strings.Contains(got, "s3cr3t-pw") {
		t.Fatalf("the password survived redaction: %q", got)
	}
	// What somebody actually wants to know must survive.
	for _, want := range []string{"db.example.com", "appdb", "appuser", "postgres"} {
		if !strings.Contains(got, want) {
			t.Errorf("redacted URL lost %q: %q", want, got)
		}
	}
}

func TestRedactValueLeavesOrdinaryValuesAlone(t *testing.T) {
	cases := map[string]any{
		"host":     "127.0.0.1",
		"port":     5432,
		"debug":    true,
		"app_url":  "https://example.com",
		"lifetime": "120m",
	}
	for key, value := range cases {
		if got := RedactValue(key, value); got != value {
			t.Errorf("RedactValue(%q, %v) = %v, want it unchanged", key, value, got)
		}
	}
}

// A shallow pass would report the whole connections block as one value and
// leak the password inside it.
func TestRedactMapWalksNestedConfiguration(t *testing.T) {
	in := map[string]any{
		"default": "pgsql",
		"connections": map[string]any{
			"pgsql": map[string]any{
				"host":     "localhost",
				"database": "appdb",
				"password": "s3cr3t-pw",
			},
		},
	}

	out := RedactMap(in)
	pgsql := out["connections"].(map[string]any)["pgsql"].(map[string]any)

	if pgsql["password"] != Redacted {
		t.Errorf("nested password = %v, want %q", pgsql["password"], Redacted)
	}
	if pgsql["host"] != "localhost" || pgsql["database"] != "appdb" {
		t.Errorf("redaction damaged the non-secret keys: %v", pgsql)
	}
	if out["default"] != "pgsql" {
		t.Errorf("top-level value changed: %v", out["default"])
	}
}

// Nothing in the output may contain the input's secrets, whatever shape the
// tree is. This is the assertion that matters, so it is made over the rendered
// whole rather than key by key.
func TestRedactMapLeaksNothingAnywhere(t *testing.T) {
	in := map[string]any{
		"app":      map[string]any{"key": "base64:AAAABBBBCCCC"},
		"jwt":      map[string]any{"secret": "jwt-signing-secret"},
		"url":      "postgres://u:db-password@host/db",
		"nested":   []any{map[string]any{"token": "bearer-token"}},
		"harmless": "keep me",
	}

	rendered := renderTree(RedactMap(in))
	for _, leaked := range []string{
		"base64:AAAABBBBCCCC", "jwt-signing-secret", "db-password", "bearer-token",
	} {
		if strings.Contains(rendered, leaked) {
			t.Errorf("secret %q appears in the redacted output: %s", leaked, rendered)
		}
	}
	if !strings.Contains(rendered, "keep me") {
		t.Errorf("redaction removed a harmless value: %s", rendered)
	}
}

func TestRedactDSNKeepsTheUsefulPartsOfBothForms(t *testing.T) {
	url := RedactDSN("postgres://appuser:pw@host.example:5432/appdb?sslmode=require")
	if strings.Contains(url, "pw@") {
		t.Errorf("URL form kept the password: %q", url)
	}
	if !strings.Contains(url, "host.example") || !strings.Contains(url, "appdb") {
		t.Errorf("URL form lost the host or database: %q", url)
	}

	kv := RedactDSN("host=localhost user=tanmay password=hunter2 dbname=appdb sslmode=disable")
	if strings.Contains(kv, "hunter2") {
		t.Errorf("keyword form kept the password: %q", kv)
	}
	for _, want := range []string{"host=localhost", "dbname=appdb", "sslmode=disable"} {
		if !strings.Contains(kv, want) {
			t.Errorf("keyword form lost %q: %q", want, kv)
		}
	}
}

func TestRedactDSNHandlesAnEmptyString(t *testing.T) {
	if got := RedactDSN(""); got != "" {
		t.Errorf("RedactDSN(\"\") = %q, want empty", got)
	}
}

// renderTree flattens a redacted tree to one string, so a leak test can assert
// over everything at once instead of walking it by hand. fmt rather than a
// type switch: the point is that *nothing* anywhere contains a secret, so the
// rendering must not be able to skip a value it did not expect.
func renderTree(in map[string]any) string {
	return fmt.Sprintf("%v", in)
}
