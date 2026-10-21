package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/ownkube/okctl/internal/api"
	"github.com/ownkube/okctl/internal/client"
	"github.com/ownkube/okctl/internal/manifest"
)

// An app's settings are one config block in the update body (appConfig,
// jobConfig, ...). An update replaces the whole block, so apply always sends
// the live block with ownkube.yaml laid over it: keys the file leaves out keep
// their live values, and keys the file never holds are echoed back untouched.

// heldBack are block keys ownkube.yaml never holds, each with where to change
// it instead. Apply echoes their live values.
var heldBack = map[string]string{
	"env":         "set variables under env at the top level of ownkube.yaml",
	"maintenance": "use 'okctl deploy maintenance'",
	"auth":        "change it in the dashboard; it holds a password",
	"persistence": "it's fixed when the app is created",
	"imageSource": "it's set by okctl from how the app is built",
	"source":      "use 'okctl deploy source'",
	"awsRoleArn":  "change it in the dashboard",
}

// topLevel maps the manifest's own top-level fields to the block keys they set.
var topLevel = map[string]string{
	"repository":   "image",
	"tag":          "tag",
	"port":         "port",
	"replicas":     "replicas",
	"public":       "public",
	"startCommand": "startCommand",
}

// spec is how one app's settings map to ownkube.yaml.
type spec struct {
	resourceType string
	key          string          // the config block in the update body
	allowed      map[string]bool // block keys ownkube.yaml may set
	hasVars      bool            // the app type has variables
	managed      bool            // Ownkube builds the image: repository and tag aren't settable
}

func specFor(d *api.Deployment) (spec, map[string]any, error) {
	rt := string(d.ResourceType)
	sp := spec{resourceType: rt}
	switch rt {
	case "web", "worker":
		sp.key, sp.hasVars = "appConfig", true
	case "job":
		sp.key, sp.hasVars = "jobConfig", true
	case "database":
		sp.key = "databaseConfig"
	case "cache":
		sp.key = "cacheConfig"
	case "function":
		return sp, nil, fmt.Errorf("%s is a function: manage it with 'okctl functions source' and 'okctl functions deploy'", d.Name)
	default:
		return sp, nil, fmt.Errorf("config doesn't manage %s apps", rt)
	}

	live, ok := normalize(d.AppConfig).(map[string]any)
	if !ok {
		return sp, nil, fmt.Errorf("%s has no settings to manage yet; try again after its first deploy", d.Name)
	}
	sp.managed = live["imageSource"] == "managed"
	sp.allowed = map[string]bool{}
	for k := range blockKeys(sp.key) {
		if _, held := heldBack[k]; held {
			continue
		}
		if sp.managed && (k == "repository" || k == "tag") {
			continue
		}
		sp.allowed[k] = true
	}
	return sp, live, nil
}

// blockKeys lists the keys the update body accepts in the named config block,
// read off the generated type so it follows the API spec.
func blockKeys(key string) map[string]bool {
	keys := map[string]bool{}
	t := reflect.TypeOf(api.UpdateDeploymentBody{})
	for i := 0; i < t.NumField(); i++ {
		if jsonName(t.Field(i)) != key {
			continue
		}
		ft := t.Field(i).Type
		for ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		if ft.Kind() != reflect.Struct {
			return keys
		}
		for j := 0; j < ft.NumField(); j++ {
			if n := jsonName(ft.Field(j)); n != "" && n != "-" {
				keys[n] = true
			}
		}
	}
	return keys
}

func jsonName(f reflect.StructField) string {
	return strings.Split(f.Tag.Get("json"), ",")[0]
}

// sensitive reports whether a settings key holds a credential. Those never go
// into ownkube.yaml (it's committed) and plan never prints their values.
func sensitive(key string) bool {
	k := strings.ToLower(key)
	for _, s := range []string{"password", "secret", "token", "credential"} {
		if strings.Contains(k, s) {
			return true
		}
	}
	return k == "barmanobjectstore"
}

// scrub drops sensitive keys from v, at any depth.
func scrub(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, x := range t {
			if !sensitive(k) {
				out[k] = scrub(x)
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = scrub(x)
		}
		return out
	}
	return v
}

func findSensitive(path string, v any) string {
	switch t := v.(type) {
	case map[string]any:
		for _, k := range sortedKeys(t) {
			if sensitive(k) {
				return join(path, k)
			}
			if p := findSensitive(join(path, k), t[k]); p != "" {
				return p
			}
		}
	case []any:
		for i, x := range t {
			if p := findSensitive(fmt.Sprintf("%s[%d]", path, i), x); p != "" {
				return p
			}
		}
	}
	return ""
}

// normalize round-trips v through JSON so YAML and API values compare equal
// (every number a float64, every map a map[string]any).
func normalize(v any) any {
	if v == nil {
		return nil
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if json.Unmarshal(raw, &out) != nil {
		return v
	}
	return out
}

// toManifest renders the live settings as ownkube.yaml. Secret variables keep
// only their names.
func toManifest(d *api.Deployment, sp spec, live map[string]any, vars []api.AppVariable) *manifest.Manifest {
	m := &manifest.Manifest{Name: d.Name, Type: sp.resourceType}
	cfg := map[string]any{}
	for _, k := range sortedKeys(live) {
		v := live[k]
		if !sp.allowed[k] || v == nil {
			continue
		}
		if !setTopLevel(m, k, v) {
			cfg[k] = scrub(v)
		}
	}
	if len(cfg) > 0 {
		m.Config = cfg
	}
	for _, v := range vars {
		if v.Source != "app" {
			continue
		}
		e := manifest.EnvVar{Name: v.Name, Secret: v.Secret}
		if !v.Secret {
			e.Value = v.Value
		}
		m.Env = append(m.Env, e)
	}
	sort.Slice(m.Env, func(i, j int) bool { return m.Env[i].Name < m.Env[j].Name })
	return m
}

// setTopLevel sets the manifest field for block key k. It reports false when
// k has no top-level field or v isn't the expected type.
func setTopLevel(m *manifest.Manifest, k string, v any) bool {
	switch k {
	case "repository", "tag", "startCommand":
		s, ok := v.(string)
		if !ok {
			return false
		}
		switch k {
		case "repository":
			m.Image = s
		case "tag":
			m.Tag = s
		default:
			m.StartCommand = s
		}
	case "port", "replicas":
		f, ok := v.(float64)
		if !ok || f != float64(int(f)) {
			return false
		}
		n := int(f)
		if k == "port" {
			m.Port = n
		} else {
			m.Replicas = &n
		}
	case "public":
		b, ok := v.(bool)
		if !ok {
			return false
		}
		m.Public = &b
	default:
		return false
	}
	return true
}

// desiredBlock is the part of the config block ownkube.yaml sets.
func desiredBlock(m *manifest.Manifest, sp spec) (map[string]any, error) {
	if m.Type != "" && m.Type != sp.resourceType {
		return nil, fmt.Errorf("%s describes a %s app, but this app is a %s", manifest.FileName, m.Type, sp.resourceType)
	}
	out := map[string]any{}
	for _, k := range sortedKeys(m.Config) {
		if !sp.allowed[k] {
			return nil, fmt.Errorf("%s: config.%s can't be set here: %s", manifest.FileName, k, notAllowed(k, sp))
		}
		if field, ok := topLevel[k]; ok {
			return nil, fmt.Errorf("%s: set %s at the top level, not under config", manifest.FileName, field)
		}
		out[k] = m.Config[k]
	}

	set := func(key string, v any) error {
		if !sp.allowed[key] {
			return fmt.Errorf("%s: %s can't be set here: %s", manifest.FileName, topLevel[key], notAllowed(key, sp))
		}
		out[key] = v
		return nil
	}
	var errs []error
	if m.Image != "" {
		errs = append(errs, set("repository", m.Image))
	}
	if m.Tag != "" {
		errs = append(errs, set("tag", m.Tag))
	}
	if m.Port != 0 {
		errs = append(errs, set("port", m.Port))
	}
	if m.Replicas != nil {
		errs = append(errs, set("replicas", *m.Replicas))
	}
	if m.Public != nil {
		errs = append(errs, set("public", *m.Public))
	}
	if m.StartCommand != "" {
		errs = append(errs, set("startCommand", m.StartCommand))
	}
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	if p := findSensitive("", out); p != "" {
		return nil, fmt.Errorf("%s is committed, so it can't hold %s; change that in the dashboard", manifest.FileName, displayPath(p))
	}
	return normalize(out).(map[string]any), nil
}

func notAllowed(key string, sp spec) string {
	if why, ok := heldBack[key]; ok {
		return why
	}
	if sp.managed && (key == "repository" || key == "tag") {
		return "Ownkube builds this app's image"
	}
	return fmt.Sprintf("it isn't a %s setting", sp.resourceType)
}

// overlay lays want over live: maps merge key by key, anything else replaces.
func overlay(live, want map[string]any) map[string]any {
	out := make(map[string]any, len(live))
	for k, v := range live {
		out[k] = v
	}
	for k, w := range want {
		lm, lok := out[k].(map[string]any)
		wm, wok := w.(map[string]any)
		if lok && wok {
			out[k] = overlay(lm, wm)
		} else {
			out[k] = w
		}
	}
	return out
}

// change is one setting that differs between live and ownkube.yaml.
type change struct {
	Path string `json:"path" yaml:"path"`
	Op   string `json:"op" yaml:"op"` // add | change
	Old  any    `json:"old,omitempty" yaml:"old,omitempty"`
	New  any    `json:"new,omitempty" yaml:"new,omitempty"`
}

func diff(path string, live, want any, out *[]change) {
	om, ook := live.(map[string]any)
	nm, nok := want.(map[string]any)
	if ook && nok {
		keys := map[string]bool{}
		for k := range om {
			keys[k] = true
		}
		for k := range nm {
			keys[k] = true
		}
		names := make([]string, 0, len(keys))
		for k := range keys {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, k := range names {
			diff(join(path, k), om[k], nm[k], out)
		}
		return
	}
	if reflect.DeepEqual(live, want) {
		return
	}
	c := change{Path: displayPath(path), Op: "change", Old: live, New: want}
	if live == nil {
		c.Op = "add"
	}
	if sensitive(lastKey(path)) {
		c.Old, c.New = nil, nil
	}
	*out = append(*out, c)
}

// varChange is one variable apply sets or removes. Value is never printed.
type varChange struct {
	Name    string `json:"name" yaml:"name"`
	Op      string `json:"op" yaml:"op"` // add | change | unset
	Secret  bool   `json:"secret,omitempty" yaml:"secret,omitempty"`
	Detail  string `json:"detail,omitempty" yaml:"detail,omitempty"`
	Problem string `json:"problem,omitempty" yaml:"problem,omitempty"`
	Value   string `json:"-" yaml:"-"`
}

// diffVars compares ownkube.yaml's env with the app's variables. An env list
// the file leaves out isn't managed; one it lists is the app's full set, so a
// variable missing from it is removed. Shared environment variables are never
// removed: listing one overrides it for this app.
func diffVars(want []manifest.EnvVar, vars []api.AppVariable) ([]varChange, error) {
	if want == nil {
		return nil, nil
	}
	own := map[string]api.AppVariable{}
	shared := map[string]api.AppVariable{}
	for _, v := range vars {
		if v.Source == "app" {
			own[v.Name] = v
		} else {
			shared[v.Name] = v
		}
	}

	var out []varChange
	seen := map[string]bool{}
	for _, e := range want {
		if e.Name == "" {
			return nil, fmt.Errorf("%s: an env entry has no name", manifest.FileName)
		}
		if seen[e.Name] {
			return nil, fmt.Errorf("%s: env %s is listed twice", manifest.FileName, e.Name)
		}
		seen[e.Name] = true
		if e.Secret && e.Value != "" {
			return nil, fmt.Errorf("%s is committed, so it can't hold the value of secret %s; set it with 'okctl vars set %s=<value> --secret' and keep only the name", manifest.FileName, e.Name, e.Name)
		}

		live, ok := own[e.Name]
		if !ok {
			live, ok = shared[e.Name]
		}
		switch {
		case !ok && e.Secret:
			out = append(out, varChange{Name: e.Name, Op: "add", Secret: true,
				Problem: fmt.Sprintf("a new secret needs a value: run 'okctl vars set %s=<value> --secret'", e.Name)})
		case !ok:
			out = append(out, varChange{Name: e.Name, Op: "add", Value: e.Value, Detail: quote(e.Value)})
		case e.Secret && live.Secret:
			// A secret's value can't be read, so a listed secret only has to exist.
		case e.Secret:
			c := varChange{Name: e.Name, Op: "change", Secret: true, Value: live.Value, Detail: "plain -> secret"}
			if live.Value == "" {
				c.Problem = "it's empty, and a secret needs a value"
			}
			out = append(out, c)
		case live.Secret:
			out = append(out, varChange{Name: e.Name, Op: "change", Value: e.Value, Detail: "secret -> plain " + quote(e.Value)})
		case live.Value != e.Value:
			out = append(out, varChange{Name: e.Name, Op: "change", Value: e.Value, Detail: quote(live.Value) + " -> " + quote(e.Value)})
		}
	}
	for _, name := range sortedNames(own) {
		if !seen[name] {
			out = append(out, varChange{Name: name, Op: "unset", Secret: own[name].Secret})
		}
	}
	return out, nil
}

// plan is the reviewed difference between an app and ownkube.yaml.
type plan struct {
	DeploymentID string      `json:"deploymentId" yaml:"deploymentId"`
	Name         string      `json:"name" yaml:"name"`
	Changes      []change    `json:"changes" yaml:"changes"`
	Variables    []varChange `json:"variables" yaml:"variables"`
	ID           string      `json:"plan" yaml:"plan"`
	InSync       bool        `json:"inSync" yaml:"inSync"`

	key   string
	block map[string]any // the full block to send when Changes is non-empty
}

// makePlan compares the app's live settings and variables with m.
func makePlan(ctx context.Context, cl *client.Client, depID string, m *manifest.Manifest) (*plan, error) {
	d, err := cl.GetDeployment(ctx, depID)
	if err != nil {
		return nil, err
	}
	sp, live, err := specFor(d)
	if err != nil {
		return nil, err
	}
	want, err := desiredBlock(m, sp)
	if err != nil {
		return nil, err
	}

	p := &plan{DeploymentID: d.Id, Name: d.Name, Changes: []change{}, Variables: []varChange{}, key: sp.key}
	p.block = overlay(live, want)
	if sp.managed {
		// An Ownkube-built image keeps its own repository and tag.
		delete(p.block, "repository")
		delete(p.block, "tag")
	}
	diff("", live, overlay(live, want), &p.Changes)

	if m.Env != nil && !sp.hasVars {
		return nil, fmt.Errorf("%s: a %s has no variables; remove env", manifest.FileName, sp.resourceType)
	}
	if sp.hasVars && m.Env != nil {
		vars, err := cl.ListVariables(ctx, d.Id)
		if err != nil {
			return nil, err
		}
		vc, err := diffVars(m.Env, vars)
		if err != nil {
			return nil, err
		}
		p.Variables = append(p.Variables, vc...)
	}

	p.InSync = len(p.Changes) == 0 && len(p.Variables) == 0
	p.ID = p.hash()
	return p, nil
}

// hash identifies exactly what apply would do, so `apply --plan` can refuse
// when the app or ownkube.yaml has moved since the plan was reviewed.
func (p *plan) hash() string {
	type hv struct {
		varChange
		Value string `json:"value"`
	}
	vars := make([]hv, len(p.Variables))
	for i, v := range p.Variables {
		vars[i] = hv{v, v.Value}
	}
	raw, _ := json.Marshal(struct {
		ID      string   `json:"id"`
		Changes []change `json:"changes"`
		Vars    []hv     `json:"vars"`
	}{p.DeploymentID, p.Changes, vars})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])[:12]
}

func (p *plan) problems() []string {
	var out []string
	for _, v := range p.Variables {
		if v.Problem != "" {
			out = append(out, fmt.Sprintf("env %s: %s", v.Name, v.Problem))
		}
	}
	return out
}

// lines renders the plan for a terminal.
func (p *plan) lines() []string {
	var out []string
	for _, c := range p.Changes {
		switch {
		case c.Old == nil && c.New == nil:
			out = append(out, fmt.Sprintf("  ~ %s (sensitive)", c.Path))
		case c.Op == "add":
			out = append(out, fmt.Sprintf("  + %s: %s", c.Path, show(c.New)))
		default:
			out = append(out, fmt.Sprintf("  ~ %s: %s -> %s", c.Path, show(c.Old), show(c.New)))
		}
	}
	for _, v := range p.Variables {
		var line string
		switch v.Op {
		case "add":
			line = "  + env " + v.Name
			if v.Secret {
				line += " (secret)"
			} else {
				line += " = " + v.Detail
			}
		case "unset":
			line = "  - env " + v.Name
		default:
			line = "  ~ env " + v.Name + ": " + v.Detail
		}
		if v.Problem != "" {
			line += "  [can't apply: " + v.Problem + "]"
		}
		out = append(out, line)
	}
	return out
}

// displayPath names a setting the way ownkube.yaml does.
func displayPath(path string) string {
	head, rest, _ := strings.Cut(path, ".")
	if field, ok := topLevel[head]; ok {
		head = field
	} else {
		head = "config." + head
	}
	if rest == "" {
		return head
	}
	return head + "." + rest
}

func show(v any) string {
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(raw)
}

func quote(s string) string { return show(s) }

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func lastKey(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[i+1:]
	}
	return path
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedNames(m map[string]api.AppVariable) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
