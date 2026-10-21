package config

import (
	"strings"
	"testing"

	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/manifest"
)

func webDeployment(cfg map[string]any) *api.Deployment {
	return &api.Deployment{Id: "dep_1", Name: "api", ResourceType: "web", AppConfig: cfg}
}

func liveWeb() map[string]any {
	return map[string]any{
		"repository":  "ghcr.io/acme/api",
		"tag":         "v1",
		"port":        8080,
		"replicas":    2,
		"public":      true,
		"env":         []any{map[string]any{"name": "TOKEN", "value": "", "secret": true}},
		"auth":        map[string]any{"basicAuth": map[string]any{"enabled": true, "username": "u", "password": "p"}},
		"resources":   map[string]any{"limits": map[string]any{"cpu": "500m", "memory": "512Mi"}},
		"maintenance": map[string]any{"enabled": false},
	}
}

func TestToManifestLeavesOutHeldBackAndSecrets(t *testing.T) {
	d := webDeployment(liveWeb())
	sp, live, err := specFor(d)
	if err != nil {
		t.Fatal(err)
	}
	vars := []api.AppVariable{
		{Name: "LOG_LEVEL", Value: "info", Source: "app"},
		{Name: "TOKEN", Secret: true, Source: "app"},
		{Name: "SHARED", Value: "x", Source: "environment"},
	}
	m := toManifest(d, sp, live, vars)
	if m.Image != "ghcr.io/acme/api" || m.Port != 8080 || m.Replicas == nil || *m.Replicas != 2 || m.Public == nil || !*m.Public {
		t.Fatalf("top-level fields: %+v", m)
	}
	for _, k := range []string{"env", "auth", "maintenance", "repository", "port"} {
		if _, ok := m.Config[k]; ok {
			t.Errorf("config holds %s", k)
		}
	}
	if _, ok := m.Config["resources"]; !ok {
		t.Errorf("config is missing resources: %v", m.Config)
	}
	if len(m.Env) != 2 || m.Env[0].Name != "LOG_LEVEL" || m.Env[1].Name != "TOKEN" || !m.Env[1].Secret || m.Env[1].Value != "" {
		t.Fatalf("env: %+v", m.Env)
	}
	data, err := m.YAML()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "password") {
		t.Fatalf("pulled yaml holds a password:\n%s", data)
	}
}

func TestOverlayKeepsWhatTheFileLeavesOut(t *testing.T) {
	sp, live, err := specFor(webDeployment(liveWeb()))
	if err != nil {
		t.Fatal(err)
	}
	replicas := 3
	m := &manifest.Manifest{
		Name:     "api",
		Replicas: &replicas,
		Config:   map[string]any{"resources": map[string]any{"limits": map[string]any{"cpu": "1"}}},
	}
	want, err := desiredBlock(m, sp)
	if err != nil {
		t.Fatal(err)
	}
	block := overlay(live, want)
	var changes []change
	diff("", live, block, &changes)
	if len(changes) != 2 {
		t.Fatalf("changes: %+v", changes)
	}
	if changes[0].Path != "replicas" || changes[1].Path != "config.resources.limits.cpu" {
		t.Fatalf("paths: %+v", changes)
	}
	limits := block["resources"].(map[string]any)["limits"].(map[string]any)
	if limits["memory"] != "512Mi" {
		t.Errorf("memory limit dropped: %v", limits)
	}
	// Keys the file never holds go back as they are.
	for _, k := range []string{"env", "auth", "maintenance", "port", "public"} {
		if _, ok := block[k]; !ok {
			t.Errorf("block dropped %s", k)
		}
	}
}

func TestDesiredBlockRejects(t *testing.T) {
	sp, _, err := specFor(webDeployment(liveWeb()))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]*manifest.Manifest{
		"held back":   {Name: "api", Config: map[string]any{"auth": map[string]any{}}},
		"top level":   {Name: "api", Config: map[string]any{"port": 80}},
		"unknown":     {Name: "api", Config: map[string]any{"nope": 1}},
		"wrong type":  {Name: "api", Type: "job"},
		"credentials": {Name: "api", Config: map[string]any{"healthCheck": map[string]any{"apiToken": "x"}}},
	}
	for name, m := range cases {
		if _, err := desiredBlock(m, sp); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestManagedImageKeepsRepository(t *testing.T) {
	cfg := liveWeb()
	cfg["imageSource"] = "managed"
	sp, _, err := specFor(webDeployment(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := desiredBlock(&manifest.Manifest{Name: "api", Image: "x"}, sp); err == nil {
		t.Fatal("image accepted on a managed build")
	}
}

func TestDiffVars(t *testing.T) {
	vars := []api.AppVariable{
		{Name: "A", Value: "1", Source: "app"},
		{Name: "B", Value: "2", Source: "app"},
		{Name: "S", Secret: true, Source: "app"},
		{Name: "OLD", Value: "x", Source: "app"},
		{Name: "SHARED", Value: "s", Source: "environment"},
	}
	want := []manifest.EnvVar{
		{Name: "A", Value: "1"},
		{Name: "B", Value: "3"},
		{Name: "S", Secret: true},
		{Name: "NEW", Value: "n"},
		{Name: "NEWSECRET", Secret: true},
	}
	got, err := diffVars(want, vars)
	if err != nil {
		t.Fatal(err)
	}
	ops := map[string]varChange{}
	for _, c := range got {
		ops[c.Name] = c
	}
	if len(ops) != 4 {
		t.Fatalf("changes: %+v", got)
	}
	if ops["B"].Op != "change" || ops["B"].Value != "3" {
		t.Errorf("B: %+v", ops["B"])
	}
	if ops["NEW"].Op != "add" || ops["NEW"].Value != "n" {
		t.Errorf("NEW: %+v", ops["NEW"])
	}
	if ops["NEWSECRET"].Problem == "" {
		t.Errorf("a new secret without a value should be flagged: %+v", ops["NEWSECRET"])
	}
	if ops["OLD"].Op != "unset" {
		t.Errorf("OLD: %+v", ops["OLD"])
	}
	if _, ok := ops["SHARED"]; ok {
		t.Error("a shared variable was unset")
	}

	if _, err := diffVars([]manifest.EnvVar{{Name: "S", Secret: true, Value: "leak"}}, vars); err == nil {
		t.Error("a secret value in the file was accepted")
	}
	if got, _ := diffVars(nil, vars); got != nil {
		t.Error("an unlisted env should leave variables alone")
	}
}

func TestPlanHashFollowsChanges(t *testing.T) {
	a := &plan{DeploymentID: "dep_1", Changes: []change{{Path: "replicas", Op: "change", Old: 2.0, New: 3.0}}}
	b := &plan{DeploymentID: "dep_1", Changes: []change{{Path: "replicas", Op: "change", Old: 2.0, New: 4.0}}}
	if a.hash() == b.hash() {
		t.Fatal("different plans share an id")
	}
	c := &plan{DeploymentID: "dep_1", Variables: []varChange{{Name: "A", Op: "change", Value: "1"}}}
	d := &plan{DeploymentID: "dep_1", Variables: []varChange{{Name: "A", Op: "change", Value: "2"}}}
	if c.hash() == d.hash() {
		t.Fatal("variable values don't feed the plan id")
	}
}
