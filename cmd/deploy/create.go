package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ownkube/okctl/cmd/internal/ux"
	"github.com/ownkube/okctl/internal/gitrepo"
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
	Github        *webRepo  `json:"github,omitempty"`
	Config        webConfig `json:"config"`
}

// webRepo is the create body's `github` member: build the image from a
// connected GitHub or GitLab repository and deploy on pushes to branch.
type webRepo struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Branch string `json:"branch"`
}

type webConfig struct {
	// Repository and Tag name the image; both are omitted for a repo build.
	Repository string      `json:"repository,omitempty"`
	Tag        string      `json:"tag,omitempty"`
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
		repo        string
		branch      string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a deployment",
		Long: "Create a deployment from a manifest file (-f) or, for a web service, " +
			"from convenience flags. Use '-f -' to read the manifest from stdin.\n\n" +
			"The manifest is the full create body (JSON or YAML) with a top-level " +
			"resourceType of web, worker, job, database, or function.\n\n" +
			"--repo builds the web service from a connected GitHub or GitLab " +
			"repository instead of an image, and deploys every push to --branch. " +
			"Either one alone is filled in from this checkout's origin remote and " +
			"current branch.",
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
				if cmd.Flags().Changed("repo") || cmd.Flags().Changed("branch") {
					dir, err := os.Getwd()
					if err != nil {
						return err
					}
					if repo, branch, err = ux.ResolveRepoBranch(cmd.Context(), dir, repo, branch); err != nil {
						return err
					}
				}
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
					repo:        repo,
					branch:      branch,
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
	cmd.Flags().StringVar(&repo, "repo", "", "Build from this repository (owner/repo) instead of --image; default with --branch: this checkout's origin")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch to build and deploy on push (with --repo); default: the current branch")
	return cmd
}

type webCreateFlags struct {
	name, image, tag     string
	environment, cluster string
	registry             string
	repo, branch         string
	port                 int
	public               bool
	envVars              []string
}

func buildWebCreateBody(f webCreateFlags) ([]byte, error) {
	var github *webRepo
	if f.repo != "" {
		if f.image != "" || f.tag != "" {
			return nil, fmt.Errorf("--repo builds the image, so it can't be combined with --image or --tag")
		}
		if f.name == "" || f.port == 0 || f.branch == "" {
			return nil, fmt.Errorf("--name, --port, and --branch are required with --repo")
		}
		owner, name, ok := gitrepo.SplitRepository(f.repo)
		if !ok {
			return nil, fmt.Errorf("--repo must look like owner/repo, got %q", f.repo)
		}
		github = &webRepo{Owner: owner, Repo: name, Branch: f.branch}
	} else if f.name == "" || f.image == "" || f.tag == "" || f.port == 0 {
		return nil, fmt.Errorf("--name, --image, --tag, and --port are required (or --repo instead of --image/--tag, or a manifest with -f)")
	}

	env, err := parseEnvVars(f.envVars)
	if err != nil {
		return nil, err
	}

	body := webCreateBody{
		ResourceType: "web",
		Name:         f.name,
		Github:       github,
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
