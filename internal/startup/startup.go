// Package startup manages an explicitly enabled per-user Ladder service.
package startup

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const label = "io.ladder.reader"

type Runner interface {
	Run(context.Context, string, ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Options provides dependencies so tests never contact the host service manager.
type Options struct {
	OS         string
	Home       string
	Executable string
	Port       int
	UID        int
	Runner     Runner
}

type State struct {
	Supported bool   `json:"supported"`
	Installed bool   `json:"installed"`
	Enabled   bool   `json:"enabled"`
	Running   bool   `json:"running"`
	Manager   string `json:"manager"`
	Path      string `json:"path"`
	Detail    string `json:"detail,omitempty"`
}

func defaults(o Options) (Options, error) {
	if o.OS == "" {
		o.OS = runtime.GOOS
	}
	if o.Home == "" {
		var err error
		o.Home, err = os.UserHomeDir()
		if err != nil {
			return o, err
		}
	}
	if !filepath.IsAbs(o.Home) {
		return o, errors.New("startup home must be an absolute path")
	}
	if o.Port == 0 {
		o.Port = 8080
	}
	if o.Port < 1 || o.Port > 65535 {
		return o, errors.New("port must be between 1 and 65535")
	}
	if o.Runner == nil {
		o.Runner = execRunner{}
	}
	if o.UID == 0 {
		o.UID = os.Getuid()
	}
	return o, nil
}

func location(o Options) (State, error) {
	s := State{Supported: true}
	switch o.OS {
	case "darwin":
		s.Manager, s.Path = "launchd", filepath.Join(o.Home, "Library", "LaunchAgents", label+".plist")
	case "linux":
		s.Manager, s.Path = "systemd --user", filepath.Join(o.Home, ".config", "systemd", "user", "ladder.service")
	default:
		s.Supported = false
		return s, fmt.Errorf("automatic startup is unsupported on %s; run ladder serve manually", o.OS)
	}
	return s, nil
}

// Status checks the actual per-user manager as well as the configuration file.
func Status(options Options) (State, error) {
	o, err := defaults(options)
	if err != nil {
		return State{}, err
	}
	s, err := location(o)
	if err != nil {
		return s, err
	}
	if _, err := os.Stat(s.Path); err == nil {
		s.Installed = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return s, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if o.OS == "darwin" {
		output, runErr := o.Runner.Run(ctx, "launchctl", "print", fmt.Sprintf("gui/%d/%s", o.UID, label))
		if runErr == nil {
			s.Enabled = true
			s.Running = strings.Contains(string(output), "state = running") || strings.Contains(string(output), "pid = ")
		} else {
			if unavailable(runErr) {
				return s, fmt.Errorf("query launchd: %w", runErr)
			}
			s.Detail = "launchd job is not loaded"
		}
	} else {
		output, runErr := o.Runner.Run(ctx, "systemctl", "--user", "is-enabled", "ladder.service")
		if unavailable(runErr) {
			return s, fmt.Errorf("query systemd: %w", runErr)
		}
		enabledState := strings.TrimSpace(string(output))
		s.Enabled = runErr == nil && (enabledState == "enabled" || enabledState == "enabled-runtime")
		active, activeErr := o.Runner.Run(ctx, "systemctl", "--user", "is-active", "ladder.service")
		if unavailable(activeErr) {
			return s, fmt.Errorf("query systemd: %w", activeErr)
		}
		s.Running = activeErr == nil && strings.TrimSpace(string(active)) == "active"
		if strings.Contains(string(output), "Failed to connect") || strings.Contains(string(active), "Failed to connect") {
			return s, errors.New("systemd user manager is unavailable")
		}
	}
	return s, nil
}

func unavailable(err error) bool {
	if err == nil {
		return false
	}
	var exit *exec.ExitError
	return !errors.As(err, &exit) // A manager's ordinary nonzero exit means disabled/inactive.
}

// Enable writes and loads a loopback-only service. It is never called implicitly.
func Enable(options Options) (State, error) {
	o, err := defaults(options)
	if err != nil {
		return State{}, err
	}
	s, err := location(o)
	if err != nil {
		return s, err
	}
	if o.Executable == "" {
		o.Executable, err = os.Executable()
		if err != nil {
			return s, err
		}
	}
	if !filepath.IsAbs(o.Executable) {
		return s, errors.New("startup executable must be an absolute path")
	}
	info, err := os.Stat(o.Executable)
	if err != nil {
		return s, fmt.Errorf("startup executable: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0111 == 0 {
		return s, errors.New("startup executable must be an executable regular file")
	}
	if strings.ContainsAny(o.Executable, "\x00\r\n") {
		return s, errors.New("startup executable contains a control character")
	}
	var content string
	port := strconv.Itoa(o.Port)
	if o.OS == "darwin" {
		logs := filepath.Join(o.Home, "Library", "Logs", "Ladder")
		if err := os.MkdirAll(logs, 0700); err != nil {
			return s, err
		}
		content = plist(o.Executable, port, logs)
	} else {
		content = "[Unit]\nDescription=Ladder reader\nAfter=network.target\n\n[Service]\nType=simple\nExecStart=" + unitQuote(o.Executable) + " serve --host 127.0.0.1 --port " + port + "\nEnvironment=BIND_ADDR=127.0.0.1\nEnvironment=PORT=" + port + "\nEnvironment=DO_NOT_TRACK=1\nRestart=on-failure\n\n[Install]\nWantedBy=default.target\n"
	}
	if err := writeConfig(s.Path, content); err != nil {
		return s, err
	}
	s.Installed = true
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	run := func(name string, args ...string) error {
		output, err := o.Runner.Run(ctx, name, args...)
		if err != nil {
			return fmt.Errorf("%s %s: %w (%s)", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
		}
		return nil
	}
	if o.OS == "darwin" {
		domain := fmt.Sprintf("gui/%d", o.UID)
		job := domain + "/" + label
		if err = run("launchctl", "enable", job); err != nil {
			return s, err
		}
		if _, loaded := o.Runner.Run(ctx, "launchctl", "print", job); loaded == nil {
			if err = run("launchctl", "bootout", job); err != nil {
				return s, err
			}
		}
		if err = run("launchctl", "bootstrap", domain, s.Path); err != nil {
			return s, err
		}
		if err = run("launchctl", "kickstart", job); err != nil {
			return s, err
		}
	} else {
		if err = run("systemctl", "--user", "daemon-reload"); err != nil {
			return s, err
		}
		if err = run("systemctl", "--user", "enable", "--now", "ladder.service"); err != nil {
			return s, err
		}
	}
	return Status(o)
}

// Disable unloads the managed job and removes only Ladder's own configuration.
func Disable(options Options) (State, error) {
	o, err := defaults(options)
	if err != nil {
		return State{}, err
	}
	s, err := location(o)
	if err != nil {
		return s, err
	}
	initial, err := Status(o)
	if err != nil {
		return initial, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if o.OS == "darwin" {
		job := fmt.Sprintf("gui/%d/%s", o.UID, label)
		if _, loaded := o.Runner.Run(ctx, "launchctl", "print", job); loaded == nil {
			if output, err := o.Runner.Run(ctx, "launchctl", "bootout", job); err != nil {
				return s, fmt.Errorf("unload launchd job: %w (%s)", err, strings.TrimSpace(string(output)))
			}
		}
		if output, err := o.Runner.Run(ctx, "launchctl", "disable", job); err != nil {
			return s, fmt.Errorf("disable launchd job: %w (%s)", err, strings.TrimSpace(string(output)))
		}
	} else {
		if initial.Installed || initial.Enabled || initial.Running {
			if output, err := o.Runner.Run(ctx, "systemctl", "--user", "disable", "--now", "ladder.service"); err != nil {
				return s, fmt.Errorf("disable systemd job: %w (%s)", err, strings.TrimSpace(string(output)))
			}
		}
	}
	if err := os.Remove(s.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return s, err
	}
	if o.OS == "linux" {
		if output, err := o.Runner.Run(ctx, "systemctl", "--user", "daemon-reload"); err != nil {
			return s, fmt.Errorf("reload systemd: %w (%s)", err, strings.TrimSpace(string(output)))
		}
	}
	return Status(o)
}

func writeConfig(name, value string) error {
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(name), ".ladder-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.WriteString(value); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), name)
}

func xmlText(value string) string {
	var out bytes.Buffer
	_ = xml.EscapeText(&out, []byte(value))
	return out.String()
}

func plist(executable, port, logs string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>Label</key><string>` + label + `</string>
<key>ProgramArguments</key><array><string>` + xmlText(executable) + `</string><string>serve</string><string>--host</string><string>127.0.0.1</string><string>--port</string><string>` + port + `</string></array>
<key>EnvironmentVariables</key><dict><key>BIND_ADDR</key><string>127.0.0.1</string><key>PORT</key><string>` + port + `</string><key>DO_NOT_TRACK</key><string>1</string></dict>
<key>RunAtLoad</key><true/><key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
<key>StandardOutPath</key><string>` + xmlText(filepath.Join(logs, "stdout.log")) + `</string><key>StandardErrorPath</key><string>` + xmlText(filepath.Join(logs, "stderr.log")) + `</string></dict></plist>
`
}

func unitQuote(value string) string {
	r := strings.NewReplacer("\\", "\\\\", `"`, `\"`, "%", "%%", "$", "$$")
	return `"` + r.Replace(value) + `"`
}
