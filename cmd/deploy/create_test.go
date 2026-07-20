package deploy

import (
	"encoding/json"
	"testing"
)

func TestBuildWebCreateBody_MinimalRequired(t *testing.T) {
	body, err := buildWebCreateBody(webCreateFlags{
		name:  "api",
		image: "ghcr.io/acme/api",
		tag:   "v1.0.0",
		port:  8080,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if got["resourceType"] != "web" {
		t.Errorf("resourceType = %v, want web", got["resourceType"])
	}
	if got["name"] != "api" {
		t.Errorf("name = %v, want api", got["name"])
	}
	// Optional pointers must be omitted, not sent as null/empty.
	if _, ok := got["environmentId"]; ok {
		t.Errorf("environmentId should be omitted when unset")
	}
	if _, ok := got["clusterId"]; ok {
		t.Errorf("clusterId should be omitted when unset")
	}
	cfg, ok := got["config"].(map[string]any)
	if !ok {
		t.Fatalf("config missing or wrong type: %v", got["config"])
	}
	if cfg["repository"] != "ghcr.io/acme/api" {
		t.Errorf("repository = %v", cfg["repository"])
	}
	if cfg["port"].(float64) != 8080 {
		t.Errorf("port = %v, want 8080", cfg["port"])
	}
	if _, ok := cfg["public"]; ok {
		t.Errorf("public should be omitted when false")
	}
}

func TestBuildWebCreateBody_AllFields(t *testing.T) {
	body, err := buildWebCreateBody(webCreateFlags{
		name:        "api",
		image:       "ghcr.io/acme/api",
		tag:         "v1.0.0",
		environment: "env-1",
		cluster:     "clus-1",
		registry:    "reg-1",
		port:        3000,
		public:      true,
		envVars:     []string{"FOO=bar", "EMPTY="},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("body is not valid JSON: %v", err)
	}
	if got["environmentId"] != "env-1" || got["clusterId"] != "clus-1" || got["registryId"] != "reg-1" {
		t.Errorf("scope fields not threaded: %v", got)
	}
	cfg := got["config"].(map[string]any)
	if cfg["public"] != true {
		t.Errorf("public = %v, want true", cfg["public"])
	}
	env, ok := cfg["env"].([]any)
	if !ok || len(env) != 2 {
		t.Fatalf("env = %v, want 2 entries", cfg["env"])
	}
	first := env[0].(map[string]any)
	if first["name"] != "FOO" || first["value"] != "bar" {
		t.Errorf("env[0] = %v", first)
	}
	second := env[1].(map[string]any)
	if second["name"] != "EMPTY" || second["value"] != "" {
		t.Errorf("env[1] = %v, want EMPTY=", second)
	}
}

func TestBuildWebCreateBody_MissingRequired(t *testing.T) {
	_, err := buildWebCreateBody(webCreateFlags{name: "api"})
	if err == nil {
		t.Fatal("expected error when required convenience flags are missing")
	}
}

func TestParseEnvVars_Invalid(t *testing.T) {
	cases := []string{"NOEQUALS", "=novalue"}
	for _, c := range cases {
		if _, err := parseEnvVars([]string{c}); err == nil {
			t.Errorf("parseEnvVars(%q) expected error", c)
		}
	}
}

func TestManifestToJSON_YAMLAndJSON(t *testing.T) {
	yamlIn := []byte("resourceType: worker\nname: bg\nconfig:\n  replicas: 2\n")
	out, err := manifestToJSON(yamlIn)
	if err != nil {
		t.Fatalf("yaml: unexpected error: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("yaml output is not JSON: %v", err)
	}
	if got["resourceType"] != "worker" || got["name"] != "bg" {
		t.Errorf("yaml decode mismatch: %v", got)
	}

	jsonIn := []byte(`{"resourceType":"job","name":"nightly"}`)
	out2, err := manifestToJSON(jsonIn)
	if err != nil {
		t.Fatalf("json: unexpected error: %v", err)
	}
	var got2 map[string]any
	if err := json.Unmarshal(out2, &got2); err != nil {
		t.Fatalf("json output is not JSON: %v", err)
	}
	if got2["name"] != "nightly" {
		t.Errorf("json decode mismatch: %v", got2)
	}
}
