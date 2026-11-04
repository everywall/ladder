package startup

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type manager struct {
	calls                    [][]string
	loaded, enabled, running bool
	failure                  error
}

func (m *manager) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	m.calls = append(m.calls, append([]string{name}, args...))
	if m.failure != nil {
		return nil, m.failure
	}
	for _, arg := range args {
		switch arg {
		case "print":
			if m.loaded {
				if m.running {
					return []byte("state = running\npid = 42"), nil
				}
				return []byte("state = waiting"), nil
			}
			return nil, &exec.ExitError{}
		case "is-enabled":
			if m.enabled {
				return []byte("enabled\n"), nil
			}
			return []byte("disabled\n"), &exec.ExitError{}
		case "is-active":
			if m.running {
				return []byte("active\n"), nil
			}
			return []byte("inactive\n"), &exec.ExitError{}
		case "bootstrap":
			m.loaded = true
		case "bootout":
			m.loaded = false
			m.running = false
		case "kickstart":
			m.running = true
		case "enable":
			m.enabled = true
			if name == "systemctl" {
				m.running = true
			}
		case "disable":
			m.enabled = false
			m.running = false
		}
	}
	return nil, nil
}

func TestStartupIsOptInAndManagerStateIsSeparate(t *testing.T) {
	for _, platform := range []string{"darwin", "linux"} {
		t.Run(platform, func(t *testing.T) {
			home := t.TempDir()
			m := &manager{}
			o := Options{OS: platform, Home: home, UID: 501, Runner: m}
			s, err := Status(o)
			if err != nil || s.Installed || s.Enabled || s.Running {
				t.Fatalf("default state %+v, %v", s, err)
			}
			if _, err := os.Stat(s.Path); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("status created configuration")
			}
			if err := os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(s.Path, []byte("installed only"), 0600); err != nil {
				t.Fatal(err)
			}
			s, err = Status(o)
			if err != nil || !s.Installed || s.Enabled || s.Running {
				t.Fatalf("file was mistaken for enabled job: %+v %v", s, err)
			}
		})
	}
}

func TestEnableDisableWithEscapedPathsAndInjectedManager(t *testing.T) {
	for _, platform := range []string{"darwin", "linux"} {
		t.Run(platform, func(t *testing.T) {
			home := t.TempDir()
			exe := filepath.Join(home, `Ladder & reader "$%`)
			if err := os.WriteFile(exe, []byte("fixture"), 0700); err != nil {
				t.Fatal(err)
			}
			m := &manager{}
			o := Options{OS: platform, Home: home, Executable: exe, Port: 4567, UID: 501, Runner: m}
			s, err := Enable(o)
			if err != nil || !s.Installed || !s.Enabled || !s.Running {
				t.Fatalf("enable %+v, %v", s, err)
			}
			data, err := os.ReadFile(s.Path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), "127.0.0.1") || !strings.Contains(string(data), "4567") {
				t.Fatalf("not bound to requested loopback port: %s", data)
			}
			if platform == "darwin" {
				decoder := xml.NewDecoder(strings.NewReader(string(data)))
				for {
					_, err := decoder.Token()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				if !strings.Contains(string(data), "&amp;") {
					t.Fatal("XML path was not escaped")
				}
			} else if !strings.Contains(string(data), `\"$$%%" serve --host 127.0.0.1 --port 4567`) {
				t.Fatalf("systemd executable was not escaped: %s", data)
			}
			for _, call := range m.calls {
				if call[0] == "sh" || call[0] == "bash" {
					t.Fatal("manager used shell")
				}
			}
			s, err = Disable(o)
			if err != nil || s.Installed || s.Enabled || s.Running {
				t.Fatalf("disable %+v, %v", s, err)
			}
		})
	}
}

func TestUnsupportedAndInvalidExecutableDoNotInstall(t *testing.T) {
	home := t.TempDir()
	m := &manager{}
	if _, err := Enable(Options{OS: "windows", Home: home, Runner: m}); err == nil {
		t.Fatal("unsupported platform accepted")
	}
	if _, err := Enable(Options{OS: "linux", Home: home, Executable: "relative", Runner: m}); err == nil {
		t.Fatal("relative executable accepted")
	}
	if len(m.calls) != 0 {
		t.Fatal("invalid enable contacted service manager")
	}
	if _, err := Status(Options{OS: "linux", Home: home, Runner: &manager{failure: errors.New("systemctl missing")}}); err == nil {
		t.Fatal("missing manager hidden")
	}
}
