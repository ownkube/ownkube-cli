package vars

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/api"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// secretMark stands in for a secret's value in the editor: okctl can't read
// secret values, so the line only says the secret exists.
const secretMark = "<secret>"

func editCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit [app]",
		Short: "Edit an app's variables in your editor",
		Long: "Open the app's own variables in $VISUAL or $EDITOR as NAME=VALUE lines, " +
			"then apply what you changed in one change: edited values are set, new lines " +
			"are added as plain variables, and deleted lines are removed. A secret shows " +
			"as " + secretMark + ": keep its line or delete it; set a new secret value with " +
			"'okctl vars set NAME --secret'. Variables shared by the environment are " +
			"listed as comments; adding a line with the same name overrides one for this app.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			depID := args[0]
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return fmt.Errorf("vars edit needs a terminal; use 'okctl vars set' and 'okctl vars delete' in scripts")
			}
			cl, err := ux.RequireClient()
			if err != nil {
				return err
			}
			vars, err := cl.ListVariables(cmd.Context(), depID)
			if err != nil {
				return err
			}
			sortVars(vars)

			edited, err := editInEditor(render(depID, vars))
			if err != nil {
				return err
			}
			lines, err := parseDotenv(edited)
			if err != nil {
				return fmt.Errorf("%w (nothing was changed)", err)
			}
			body, err := changes(vars, lines)
			if err != nil {
				return fmt.Errorf("%w (nothing was changed)", err)
			}
			if body.Set == nil && body.Unset == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "No changes.")
				return nil
			}
			return update(cmd, cl, depID, body)
		},
	}
}

func render(depID string, vars []api.AppVariable) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Variables for %s: one NAME=VALUE per line. Delete a line to remove it.\n", depID)
	fmt.Fprintf(&b, "# %s marks a secret, whose value okctl can't read: keep or delete the line.\n", secretMark)
	fmt.Fprintln(&b, "# Set a secret's value with: okctl vars set NAME --secret")
	var shared []api.AppVariable
	for _, v := range vars {
		if v.Source != "app" {
			shared = append(shared, v)
		}
	}
	if len(shared) > 0 {
		fmt.Fprintln(&b, "#")
		fmt.Fprintln(&b, "# Shared by the environment (a line below with the same name overrides it here):")
		for _, v := range shared {
			value := dotenvValue(v.Value)
			if v.Secret {
				value = secretMark
			}
			fmt.Fprintf(&b, "#   %s=%s\n", v.Name, value)
		}
	}
	fmt.Fprintln(&b)
	for _, v := range vars {
		if v.Source != "app" {
			continue
		}
		value := dotenvValue(v.Value)
		if v.Secret {
			value = secretMark
		}
		fmt.Fprintf(&b, "%s=%s\n", v.Name, value)
	}
	return b.String()
}

// editInEditor opens content in the user's editor and returns what they saved.
// The file only ever holds plain values; secrets appear as secretMark.
func editInEditor(content string) (string, error) {
	f, err := os.CreateTemp("", "okctl-vars-*.env")
	if err != nil {
		return "", err
	}
	path := f.Name()
	defer os.Remove(path)
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
		if runtime.GOOS == "windows" {
			editor = "notepad"
		}
	}
	parts := strings.Fields(editor)
	c := exec.Command(parts[0], append(parts[1:], path)...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return "", fmt.Errorf("editor %s: %w", parts[0], err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

type entry struct {
	value  string
	marked bool // the line is secretMark, unquoted
}

// parseDotenv reads NAME=VALUE lines, skipping blanks and # comments. A value
// in double quotes is unquoted Go-style, which is how dotenvValue writes one.
func parseDotenv(s string) (map[string]entry, error) {
	out := map[string]entry{}
	sc := bufio.NewScanner(strings.NewReader(s))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		name = strings.TrimSpace(name)
		if !ok {
			return nil, fmt.Errorf("line %d: expected NAME=VALUE", n)
		}
		if err := checkName(name); err != nil {
			return nil, fmt.Errorf("line %d: %w", n, err)
		}
		if _, dup := out[name]; dup {
			return nil, fmt.Errorf("line %d: %s is listed twice", n, name)
		}
		value = strings.TrimSpace(value)
		e := entry{value: value, marked: value == secretMark}
		if strings.HasPrefix(value, `"`) {
			v, err := strconv.Unquote(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: %s has a badly quoted value", n, name)
			}
			e = entry{value: v}
		}
		out[name] = e
	}
	return out, sc.Err()
}

// changes works out the variables update that turns vars into the edited set.
func changes(vars []api.AppVariable, edited map[string]entry) (api.UpdateAppVariablesBody, error) {
	var set []api.AppVariableInput
	var unset []string
	own := map[string]api.AppVariable{}
	for _, v := range vars {
		if v.Source == "app" {
			own[v.Name] = v
		}
	}
	for _, v := range vars {
		if v.Source != "app" {
			continue
		}
		e, kept := edited[v.Name]
		switch {
		case !kept:
			unset = append(unset, v.Name)
		case v.Secret && !e.marked:
			return api.UpdateAppVariablesBody{}, fmt.Errorf("%s is a secret, so its value can't be changed in the editor; use 'okctl vars set %s --secret'", v.Name, v.Name)
		case !v.Secret && e.marked:
			return api.UpdateAppVariablesBody{}, fmt.Errorf("%s isn't a secret; give it a value", v.Name)
		case !v.Secret && e.value != v.Value:
			set = append(set, api.AppVariableInput{Name: v.Name, Value: e.value})
		}
	}
	for _, name := range sortedEntries(edited) {
		if _, ok := own[name]; ok {
			continue
		}
		e := edited[name]
		if e.marked {
			return api.UpdateAppVariablesBody{}, fmt.Errorf("%s is new, so it needs a value; set a new secret with 'okctl vars set %s --secret'", name, name)
		}
		set = append(set, api.AppVariableInput{Name: name, Value: e.value})
	}

	var body api.UpdateAppVariablesBody
	if len(set) > 0 {
		body.Set = &set
	}
	if len(unset) > 0 {
		body.Unset = &unset
	}
	return body, nil
}

// dotenvValue writes v so parseDotenv reads it back unchanged.
func dotenvValue(v string) string {
	if v == "" {
		return ""
	}
	if v != strings.TrimSpace(v) || strings.ContainsAny(v, "\n\r") || strings.HasPrefix(v, `"`) || strings.HasPrefix(v, "#") || v == secretMark {
		return strconv.Quote(v)
	}
	return v
}

func sortedEntries(m map[string]entry) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}
