package mcp

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

// Redacted is what replaces a secret. A fixed string rather than removing the
// key, because "this value exists and is withheld" is useful to an agent and
// "this key is absent" is a different, misleading fact.
const Redacted = "<redacted>"

// secretKeyFragments are matched case-insensitively against a config key.
//
// Deny by pattern rather than allow by list, deliberately. An allowlist of
// safe keys would have to be extended for every new configuration block, and
// the failure mode of forgetting is a leaked credential — whereas the failure
// mode of over-redaction is an agent asking you for a value.
var secretKeyFragments = []string{
	"password", "passwd",
	"secret",
	"token",
	"apikey", "api_key",
	"credential",
	"private",
	"salt",
	"signature",
	"dsn",
	// "key" last and on its own, because it is the broadest: it catches
	// app.key and jwt.key, and also over-matches harmless things like
	// "foreign_key". Over-matching is the right side to err on.
	"key",
}

// IsSecretKey reports whether a configuration key should have its value
// withheld.
func IsSecretKey(key string) bool {
	lower := strings.ToLower(key)
	for _, fragment := range secretKeyFragments {
		if strings.Contains(lower, fragment) {
			return true
		}
	}
	return false
}

// RedactValue withholds a value that is secret by key, or that carries a
// credential in its own text whatever the key is called.
//
// The second case is the one a key-based rule alone gets wrong, and it is not
// hypothetical: DATABASE_URL is the standard name for a Postgres connection
// string, the name contains none of the fragments above, and the string
// contains the password. So a URL with a password in its userinfo is rewritten
// rather than withheld — the host and database stay visible, which is what
// somebody actually wants to know, and the credential does not.
func RedactValue(key string, value any) any {
	if IsSecretKey(key) {
		return Redacted
	}

	text, ok := value.(string)
	if !ok {
		return value
	}
	if redacted, changed := redactURLCredentials(text); changed {
		return redacted
	}
	return value
}

// redactURLCredentials rewrites the userinfo of a URL, reporting whether it
// changed anything.
func redactURLCredentials(text string) (string, bool) {
	// Cheap guard first: url.Parse accepts almost any string, so without this
	// every plain value would go through the parser and some would come back
	// subtly rewritten.
	if !strings.Contains(text, "://") {
		return text, false
	}
	parsed, err := url.Parse(text)
	if err != nil || parsed.User == nil {
		return text, false
	}
	if _, hasPassword := parsed.User.Password(); !hasPassword {
		return text, false
	}
	parsed.User = url.UserPassword(parsed.User.Username(), Redacted)
	// url.URL.String re-encodes the redaction's angle brackets, which is
	// correct but unreadable, so the marker is put back verbatim.
	return strings.Replace(parsed.String(), url.QueryEscape(Redacted), Redacted, 1), true
}

// asStringMap reports whether v is any map keyed by string, and returns it as
// a plain map[string]any.
//
// Reflection rather than a type assertion, because the configuration is built
// from config.M — a NAMED map type — and `v.(map[string]any)` does not match a
// named type with that underlying type. A type switch missed every nested
// subtree, so a password at sql.connections.pgsql.password passed through
// unredacted while the test suite, which used plain maps, stayed green.
//
// Found by calling the tool against a real application and reading what came
// back, which no unit test written in the same idiom as the code would have
// caught.
func asStringMap(v any) (map[string]any, bool) {
	if direct, ok := v.(map[string]any); ok {
		return direct, true
	}
	value := reflect.ValueOf(v)
	if !value.IsValid() || value.Kind() != reflect.Map {
		return nil, false
	}
	if value.Type().Key().Kind() != reflect.String {
		return nil, false
	}
	out := make(map[string]any, value.Len())
	iter := value.MapRange()
	for iter.Next() {
		out[iter.Key().String()] = iter.Value().Interface()
	}
	return out, true
}

// asSlice reports whether v is any slice or array, and returns its elements.
//
// Same reason as asStringMap: a named slice type would not match `[]any`.
func asSlice(v any) ([]any, bool) {
	if direct, ok := v.([]any); ok {
		return direct, true
	}
	value := reflect.ValueOf(v)
	if !value.IsValid() || (value.Kind() != reflect.Slice && value.Kind() != reflect.Array) {
		return nil, false
	}
	// A []byte is data, not a list to walk into.
	if value.Type().Elem().Kind() == reflect.Uint8 {
		return nil, false
	}
	out := make([]any, 0, value.Len())
	for i := 0; i < value.Len(); i++ {
		out = append(out, value.Index(i).Interface())
	}
	return out, true
}

// RedactMap walks a configuration tree and withholds every secret in it.
//
// Nested maps are walked rather than stringified, because the configuration is
// nested — sql.connections.pgsql.password is where a password actually lives,
// and a shallow pass would report the whole connections block as one opaque
// value and leak it.
func RedactMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		if IsSecretKey(key) {
			// Withheld whatever its shape: a secret key holding a map is
			// still a secret, and recursing into it would report its parts.
			out[key] = Redacted
			continue
		}
		if nested, ok := asStringMap(value); ok {
			out[key] = RedactMap(nested)
			continue
		}
		if items, ok := asSlice(value); ok {
			redacted := make([]any, 0, len(items))
			for _, item := range items {
				if nested, ok := asStringMap(item); ok {
					redacted = append(redacted, RedactMap(nested))
					continue
				}
				redacted = append(redacted, RedactValue(key, item))
			}
			out[key] = redacted
			continue
		}
		out[key] = RedactValue(key, value)
	}
	return out
}

// RedactDSN reduces a connection string to the parts worth reporting.
//
// Used where a DSN is the subject rather than an incidental value — the
// database-connections tool — so that "which database am I pointed at" is
// answerable without handing over the credential.
func RedactDSN(dsn string) string {
	if dsn == "" {
		return ""
	}
	if strings.Contains(dsn, "://") {
		parsed, err := url.Parse(dsn)
		if err == nil {
			host := parsed.Host
			database := strings.TrimPrefix(parsed.Path, "/")
			user := ""
			if parsed.User != nil {
				user = parsed.User.Username()
			}
			return fmt.Sprintf("%s://%s@%s/%s", parsed.Scheme, user, host, database)
		}
	}
	// Keyword/value form: host=... user=... password=... Report every pair
	// except the ones that are secret.
	fields := strings.Fields(dsn)
	kept := make([]string, 0, len(fields))
	for _, field := range fields {
		name, _, found := strings.Cut(field, "=")
		if found && IsSecretKey(name) {
			kept = append(kept, name+"="+Redacted)
			continue
		}
		kept = append(kept, field)
	}
	return strings.Join(kept, " ")
}
