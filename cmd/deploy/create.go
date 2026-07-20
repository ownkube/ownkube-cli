package deploy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/spf13/cobra"
)

// webCreateBody is the JSON shape for a web deployment built from convenience
// flags. It mirrors the server's `resourceType: "web"` create member. The
// manifest path (`-f`) bypasses this entirely and sends the file verbatim.
type webCreateBody struct {
	ResourceType  string    `json:"resourceType"`
	Name          string    `json:"name"`
	EnvironmentId *string   `json:"environmentId,omitempty"`
	ClusterId     *string   `json:"clusterId,omitempty"`
	RegistryId    *string   `json:"registryId,omitempty"`
	Config        webConfig `json:"config"`
}

type webConfig struct {
	Repository string      `json:"repository"`
	Tag        string      `json:"tag"`
	Port       int         `json:"port"`
	Public     *bool       `json:"public,omitempty"`
	Env        []webEnvVar `json:"env,omitempty"`
}

type webEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func createCmd() *cobra.Command {
	var (
		file        string
		name        string
		image       string
		tag         string
		environment string
		cluster     string
		registry    string
		port        int
		public      bool
		envVars     []string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a deployment",
		Long: "Create a deployment from a manifest file (-f) or, for a web service, " +
			"from convenience flags. Use '-f -' to read the manifest from stdin.\n\n" +
			"The manifest is the full create body (JSON or YAML) with a top-level " +
			"resourceType of web, worker, job, database, or function.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			api, err := ux.RequireClient()
			if err != nil {
				return err
			}

			var body []byte
			if file != "" {
				raw, err := ux.ReadFileOrStdin(file)
				if err != nil {
					return fmt.Errorf("reading manifest: %w", err)
				}
				body, err = manifestToJSON(raw)
				if err != nil {
					return fmt.Errorf("parsing manifest: %w", err)
				}
			} else {
				body, err = buildWebCreateBody(webCreateFlags{
					name:        name,
					image:       image,
					tag:         tag,
					environment: environment,
					cluster:     cluster,
					registry:    registry,
					port:        port,
					public:      public,
					envVars:     envVars,
				})
				if err != nil {
					return err
				}
			}

			d, err := api.CreateDeployment(cmd.Context(), body)
			if err != nil {
				return err
			}
			return renderCreated(cmd, d)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to a manifest file (JSON or YAML); '-' for stdin")
	cmd.Flags().StringVar(&name, "name", "", "Deployment name (web convenience form)")
	cmd.Flags().StringVar(&image, "image", "", "Container image repository (web convenience form)")
	cmd.Flags().StringVar(&tag, "tag", "", "Image tag (web convenience form)")
	cmd.Flags().StringVar(&environment, "environment", "", "Environment ID")
	cmd.Flags().StringVar(&cluster, "cluster", "", "Cluster ID (omit to use the shared cluster)")
	cmd.Flags().StringVar(&registry, "registry", "", "Registry ID for a private image")
	cmd.Flags().IntVar(&port, "port", 0, "Container port (web convenience form)")
	cmd.Flags().BoolVar(&public, "public", false, "Expose the service on a public hostname")
	cmd.Flags().StringArrayVar(&envVars, "env", nil, "Environment variable KEY=VALUE (repeatable)")
	return cmd
}

type webCreateFlags struct {
	name, image, tag     string
	environment, cluster string
	registry             string
	port                 int
	public               bool
	envVars              []string
}

func buildWebCreateBody(f webCreateFlags) ([]byte, error) {
	if f.name == "" || f.image == "" || f.tag == "" || f.port == 0 {
		return nil, fmt.Errorf("--name, --image, --tag, and --port are required (or pass a manifest with -f)")
	}

	env, err := parseEnvVars(f.envVars)
	if err != nil {
		return nil, err
	}

	body := webCreateBody{
		ResourceType: "web",
		Name:         f.name,
		Config: webConfig{
			Repository: f.image,
			Tag:        f.tag,
			Port:       f.port,
			Env:        env,
		},
	}
	if f.environment != "" {
		body.EnvironmentId = &f.environment
	}
	if f.cluster != "" {
		body.ClusterId = &f.cluster
	}
	if f.registry != "" {
		body.RegistryId = &f.registry
	}
	if f.public {
		p := true
		body.Config.Public = &p
	}
	return json.Marshal(body)
}

func parseEnvVars(pairs []string) ([]webEnvVar, error) {
	if len(pairs) == 0 {
		return nil, nil
	}
	out := make([]webEnvVar, 0, len(pairs))
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("invalid --env %q: expected KEY=VALUE", p)
		}
		out = append(out, webEnvVar{Name: k, Value: v})
	}
	return out, nil
}
