package config

import "testing"

func TestLoadCredentialsPrefersEnvKey(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SaveCredentials(&Credentials{APIKey: "file-key"}); err != nil {
		t.Fatal(err)
	}

	t.Setenv(APIKeyEnv, "okws_env")
	creds, err := m.LoadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if creds.APIKey != "okws_env" || !creds.FromEnv {
		t.Fatalf("got %+v, want the env key", creds)
	}

	t.Setenv(APIKeyEnv, "")
	creds, err = m.LoadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if creds.APIKey != "file-key" || creds.FromEnv {
		t.Fatalf("got %+v, want the file key", creds)
	}
}
