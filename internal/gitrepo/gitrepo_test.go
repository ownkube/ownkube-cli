package gitrepo

import "testing"

func TestParseRemoteURL(t *testing.T) {
	cases := []struct {
		in, host, repo string
		ok             bool
	}{
		{"git@github.com:pwnedlabs/pwnrange.git", "github.com", "pwnedlabs/pwnrange", true},
		{"git@github.com:pwnedlabs/pwnrange", "github.com", "pwnedlabs/pwnrange", true},
		{"https://github.com/pwnedlabs/pwnrange.git", "github.com", "pwnedlabs/pwnrange", true},
		{"https://github.com/pwnedlabs/pwnrange/", "github.com", "pwnedlabs/pwnrange", true},
		{"https://user:token@github.com/owner/repo.git", "github.com", "owner/repo", true},
		{"ssh://git@github.com/owner/repo.git", "github.com", "owner/repo", true},
		{"ssh://git@gitlab.example.com:2222/group/sub/project.git", "gitlab.example.com", "group/sub/project", true},
		{"git@gitlab.com:group/sub/project.git", "gitlab.com", "group/sub/project", true},
		{"GIT@GitHub.com:Owner/Repo.git", "github.com", "Owner/Repo", true},
		{"/srv/git/repo.git", "", "", false},
		{"./relative/repo", "", "", false},
		{"https://github.com/onlyowner", "", "", false},
		{"", "", "", false},
	}
	for _, c := range cases {
		host, repo, ok := ParseRemoteURL(c.in)
		if ok != c.ok || host != c.host || repo != c.repo {
			t.Errorf("ParseRemoteURL(%q) = (%q, %q, %v), want (%q, %q, %v)",
				c.in, host, repo, ok, c.host, c.repo, c.ok)
		}
	}
}

func TestSplitRepository(t *testing.T) {
	cases := []struct {
		in, owner, repo string
		ok              bool
	}{
		{"owner/repo", "owner", "repo", true},
		{"owner/repo.git", "owner", "repo", true},
		{"group/sub/project", "group/sub", "project", true},
		{"/owner/repo/", "owner", "repo", true},
		{"repo", "", "", false},
		{"owner/", "", "", false},
	}
	for _, c := range cases {
		owner, repo, ok := SplitRepository(c.in)
		if ok != c.ok || owner != c.owner || repo != c.repo {
			t.Errorf("SplitRepository(%q) = (%q, %q, %v), want (%q, %q, %v)",
				c.in, owner, repo, ok, c.owner, c.repo, c.ok)
		}
	}
}
