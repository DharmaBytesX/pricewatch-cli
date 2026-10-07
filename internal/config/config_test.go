package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestResolveOrder(t *testing.T) {
	file := File{URL: "https://file.example/", Token: "pwt_file"}
	cases := []struct {
		flagURL, flagToken string
		env                map[string]string
		want               Settings
	}{
		{"", "", nil, Settings{"https://file.example", SourceFile, "pwt_file", SourceFile}},
		{"", "", map[string]string{EnvURL: "https://env.example", EnvToken: "pwt_env"},
			Settings{"https://env.example", SourceEnv, "pwt_env", SourceEnv}},
		{"https://flag.example", "pwt_flag", map[string]string{EnvURL: "https://env.example", EnvToken: "pwt_env"},
			Settings{"https://flag.example", SourceFlag, "pwt_flag", SourceFlag}},
	}
	for _, c := range cases {
		if got := Resolve(c.flagURL, c.flagToken, env(c.env), file); got != c.want {
			t.Errorf("got %+v, want %+v", got, c.want)
		}
	}
	if got := Resolve("", "", env(nil), File{}); got.URL != DefaultURL || got.URLSource != SourceDefault || got.Token != "" {
		t.Errorf("defaults: %+v", got)
	}
}

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	if f, err := Load(path); err != nil || f != (File{}) {
		t.Fatalf("missing file: %+v %v", f, err)
	}
	want := File{URL: "https://example.com", Token: "pwt_secret"}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || got != want {
		t.Fatalf("got %+v %v", got, err)
	}
	if runtime.GOOS != "windows" {
		if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
			t.Fatalf("file mode %v: the token must be readable by its owner only", info.Mode().Perm())
		}
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("a broken file was accepted")
	}
}

func TestPath(t *testing.T) {
	if p, _ := Path(env(map[string]string{EnvConfig: "/tmp/pw.json"})); p != "/tmp/pw.json" {
		t.Fatal(p)
	}
	if p, err := Path(env(nil)); err == nil && filepath.Base(p) != "config.json" {
		t.Fatal(p)
	}
}
