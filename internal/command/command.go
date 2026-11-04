// Package command handles non-server Ladder commands before the legacy parser.
package command

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"ladder/internal/mcp"
	"ladder/internal/service"
	"ladder/internal/startup"
	"os"
	"strings"
)

type backend interface {
	mcp.Backend
	Packs() []service.PackInfo
}

type dependencies struct {
	newService     func(options) (backend, error)
	checkPacks     func([]string) ([]service.PackInfo, error)
	startupStatus  func(startup.Options) (startup.State, error)
	startupEnable  func(startup.Options) (startup.State, error)
	startupDisable func(startup.Options) (startup.State, error)
}

type paths []string

func (p *paths) String() string { return strings.Join(*p, ";") }
func (p *paths) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("pack path must not be empty")
	}
	*p = append(*p, value)
	return nil
}

type options struct {
	format  string
	json    bool
	refresh bool
	cache   string
	ruleset string
	packs   paths
	stdio   bool
}

// Run returns handled=false for legacy server arguments. args excludes argv[0].
func Run(args []string, stdin io.Reader, stdout, stderr io.Writer) (handled bool, exitCode int) {
	return run(args, stdin, stdout, stderr, dependencies{newService: construct, checkPacks: service.CheckPacks, startupStatus: startup.Status, startupEnable: startup.Enable, startupDisable: startup.Disable})
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, d dependencies) (bool, int) {
	if len(args) == 0 {
		return false, 0
	}
	switch args[0] {
	case "fetch":
		return true, fetch(args[1:], stdout, stderr, d)
	case "rules":
		return true, rules(args[1:], stdout, stderr, d)
	case "mcp":
		return true, connector(args[1:], stdin, stdout, stderr, d)
	case "startup":
		return true, startupCommand(args[1:], stdout, stderr, d)
	default:
		if strings.HasPrefix(args[0], "-") || args[0] == "serve" {
			return false, 0
		}
		fmt.Fprintf(stderr, "unknown command %q; use fetch, rules, mcp, startup, or serve\n", args[0])
		return true, 2
	}
}

func construct(o options) (backend, error) {
	if o.ruleset == "" && len(o.packs) == 0 {
		return service.NewFromEnv()
	}
	cfg, err := service.ConfigFromLookup(func(key string) (string, bool) {
		if key == "RULESET" && o.ruleset != "" {
			return o.ruleset, true
		}
		if key == "RULE_PACKS" && len(o.packs) > 0 {
			value := strings.Join(o.packs, ";")
			if existing := os.Getenv(key); existing != "" {
				value = existing + ";" + value
			}
			return value, true
		}
		return os.LookupEnv(key)
	})
	if err != nil {
		return nil, err
	}
	return service.New(cfg)
}

func flags(name string, stderr io.Writer, o *options) *flag.FlagSet {
	f := flag.NewFlagSet(name, flag.ContinueOnError)
	f.SetOutput(stderr)
	f.BoolVar(&o.json, "json", false, "write one JSON result, including metadata")
	f.StringVar(&o.ruleset, "ruleset", "", "legacy YAML ruleset path, directory, or URL")
	f.Var(&o.packs, "pack", "versioned site-pack file (repeatable)")
	return f
}

func readFlags(f *flag.FlagSet, o *options) {
	f.StringVar(&o.cache, "cache", "default", "cache mode: default, reload, bypass")
	f.BoolVar(&o.refresh, "refresh", false, "refresh the cached page (same as --cache reload)")
}

// FlagSet stops at a positional argument; reorder known flags to permit URL first.
func parse(f *flag.FlagSet, args []string) error {
	var opts, positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}
		opts = append(opts, arg)
		name := strings.TrimLeft(strings.SplitN(arg, "=", 2)[0], "-")
		entry := f.Lookup(name)
		if entry == nil || strings.Contains(arg, "=") {
			continue
		}
		boolean, ok := entry.Value.(interface{ IsBoolFlag() bool })
		if ok && boolean.IsBoolFlag() {
			continue
		}
		if i+1 < len(args) {
			i++
			opts = append(opts, args[i])
		}
	}
	return f.Parse(append(append(opts, "--"), positional...))
}

func validateCache(o *options) error {
	if o.refresh {
		if o.cache != "default" && o.cache != "reload" {
			return errors.New("--refresh conflicts with --cache bypass")
		}
		o.cache = "reload"
	}
	if o.cache != "default" && o.cache != "reload" && o.cache != "bypass" {
		return errors.New("cache must be default, reload, or bypass")
	}
	return nil
}

func parseCode(err error) int {
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	return 2
}

func fetch(args []string, stdout, stderr io.Writer, d dependencies) int {
	var o options
	f := flags("ladder fetch [flags] URL", stderr, &o)
	readFlags(f, &o)
	f.StringVar(&o.format, "format", "markdown", "output format: markdown, text, html")
	if err := parse(f, args); err != nil {
		return parseCode(err)
	}
	if f.NArg() != 1 {
		return fail(stdout, stderr, o.json, errors.New("fetch requires exactly one URL"), 2)
	}
	if o.format != "markdown" && o.format != "text" && o.format != "html" {
		return fail(stdout, stderr, o.json, errors.New("format must be markdown, text, or html"), 2)
	}
	if err := validateCache(&o); err != nil {
		return fail(stdout, stderr, o.json, err, 2)
	}
	svc, err := d.newService(o)
	if err != nil {
		return fail(stdout, stderr, o.json, err, 1)
	}
	doc, err := svc.Read(context.Background(), service.Request{URL: f.Arg(0), Cache: o.cache})
	if err != nil {
		return fail(stdout, stderr, o.json, err, errorCode(err))
	}
	content := doc.Content.Markdown
	if o.format == "text" {
		content = doc.Content.Text
	}
	if o.format == "html" {
		content = doc.Content.HTML
	}
	code := 0
	if doc.Metadata.UpstreamStatus >= 400 {
		code = 1
	}
	if o.json {
		if writeJSON(stdout, stderr, map[string]any{"metadata": doc.Metadata, "title": doc.Title, "byline": doc.Byline, "format": o.format, "content": content, "problems": doc.Problems}) != 0 {
			return 1
		}
		return code
	}
	if _, err := io.WriteString(stdout, content); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if !strings.HasSuffix(content, "\n") {
		if _, err := io.WriteString(stdout, "\n"); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if code != 0 {
		fmt.Fprintf(stderr, "upstream returned HTTP %d\n", doc.Metadata.UpstreamStatus)
	}
	return code
}

func rules(args []string, stdout, stderr io.Writer, d dependencies) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: ladder rules list|check|test [flags] [URL]")
		return 2
	}
	action := args[0]
	if action != "list" && action != "check" && action != "test" {
		if action == "--help" || action == "-h" {
			fmt.Fprintln(stdout, "usage: ladder rules list|check|test [flags] [URL]")
			return 0
		}
		fmt.Fprintln(stderr, "rules requires list, check, or test")
		return 2
	}
	var o options
	f := flags("ladder rules "+action, stderr, &o)
	if action == "test" {
		readFlags(f, &o)
	}
	if err := parse(f, args[1:]); err != nil {
		return parseCode(err)
	}
	if action == "check" {
		o.packs = append(o.packs, f.Args()...)
		// Constructing validates the configured legacy ruleset and pack composition.
		svc, err := d.newService(o)
		if err != nil {
			return fail(stdout, stderr, o.json, err, 1)
		}
		packs := svc.Packs()
		if len(o.packs) > 0 {
			if _, err = d.checkPacks(o.packs); err != nil {
				return fail(stdout, stderr, o.json, err, 1)
			}
		}
		if o.json {
			return writeJSON(stdout, stderr, map[string]any{"valid": true, "packs": packs})
		}
		fmt.Fprintf(stdout, "Rules are valid (%d versioned packs).\n", len(packs))
		return 0
	}
	if (action == "list" && f.NArg() != 0) || (action == "test" && f.NArg() != 1) {
		return fail(stdout, stderr, o.json, errors.New("rules list takes no URL; rules test requires exactly one URL"), 2)
	}
	if action == "test" {
		if err := validateCache(&o); err != nil {
			return fail(stdout, stderr, o.json, err, 2)
		}
	}
	svc, err := d.newService(o)
	if err != nil {
		return fail(stdout, stderr, o.json, err, 1)
	}
	if action == "list" {
		packs := svc.Packs()
		if packs == nil {
			packs = []service.PackInfo{}
		}
		if o.json {
			return writeJSON(stdout, stderr, map[string]any{"packs": packs})
		}
		if len(packs) == 0 {
			fmt.Fprintln(stdout, "No versioned site packs configured. Use --pack PATH or RULE_PACKS.")
			return 0
		}
		for _, p := range packs {
			fmt.Fprintf(stdout, "%s@%s\t%d rules\tsha256:%s\n", p.ID, p.Version, p.Rules, p.SHA256)
		}
		return 0
	}
	result, err := svc.TestURL(context.Background(), service.Request{URL: f.Arg(0), Cache: o.cache})
	if err != nil {
		return fail(stdout, stderr, o.json, err, errorCode(err))
	}
	code := 0
	if !result.Passed {
		code = 1
	}
	if o.json {
		if writeJSON(stdout, stderr, result) != 0 {
			return 1
		}
		return code
	}
	fmt.Fprintf(stdout, "URL: %s\n", result.Metadata.FinalURL)
	for _, check := range result.Checks {
		status := "FAIL"
		if check.Passed {
			status = "PASS"
		}
		fmt.Fprintf(stdout, "%s %s: %s\n", status, check.Name, check.Detail)
	}
	return code
}

func connector(args []string, stdin io.Reader, stdout, stderr io.Writer, d dependencies) int {
	var o options
	f := flag.NewFlagSet("ladder mcp --stdio", flag.ContinueOnError)
	f.SetOutput(stderr)
	f.BoolVar(&o.stdio, "stdio", false, "serve MCP over standard input and output")
	f.StringVar(&o.ruleset, "ruleset", "", "legacy YAML ruleset path, directory, or URL")
	f.Var(&o.packs, "pack", "versioned site-pack file (repeatable)")
	if err := parse(f, args); err != nil {
		return parseCode(err)
	}
	if !o.stdio || f.NArg() != 0 {
		fmt.Fprintln(stderr, "usage: ladder mcp --stdio [--ruleset PATH] [--pack PATH]")
		return 2
	}
	svc, err := d.newService(o)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := mcp.Serve(context.Background(), stdin, stdout, svc); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func startupCommand(args []string, stdout, stderr io.Writer, d dependencies) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: ladder startup status|enable|disable [--json] [--executable PATH] [--port PORT]")
		return 2
	}
	action := args[0]
	if action != "status" && action != "enable" && action != "disable" {
		if action == "--help" || action == "-h" {
			fmt.Fprintln(stdout, "usage: ladder startup status|enable|disable [flags]")
			return 0
		}
		fmt.Fprintln(stderr, "startup requires status, enable, or disable")
		return 2
	}
	f := flag.NewFlagSet("ladder startup "+action, flag.ContinueOnError)
	f.SetOutput(stderr)
	var o startup.Options
	var asJSON bool
	f.BoolVar(&asJSON, "json", false, "write service-manager state as JSON")
	if action == "enable" {
		f.StringVar(&o.Executable, "executable", "", "absolute path to installed Ladder executable")
		f.IntVar(&o.Port, "port", 8080, "loopback server port")
	}
	if err := parse(f, args[1:]); err != nil {
		return parseCode(err)
	}
	if f.NArg() != 0 {
		return fail(stdout, stderr, asJSON, errors.New("unexpected startup arguments"), 2)
	}
	if action == "enable" && (o.Port < 1 || o.Port > 65535) {
		return fail(stdout, stderr, asJSON, errors.New("port must be between 1 and 65535"), 2)
	}
	fn := d.startupStatus
	if action == "enable" {
		fn = d.startupEnable
	}
	if action == "disable" {
		fn = d.startupDisable
	}
	state, err := fn(o)
	if err != nil {
		return fail(stdout, stderr, asJSON, err, 1)
	}
	if asJSON {
		return writeJSON(stdout, stderr, state)
	}
	fmt.Fprintf(stdout, "Startup: enabled=%t running=%t installed=%t\nManager: %s\nConfiguration: %s\n", state.Enabled, state.Running, state.Installed, state.Manager, state.Path)
	if state.Detail != "" {
		fmt.Fprintln(stdout, state.Detail)
	}
	return 0
}

func errorCode(err error) int {
	var e *service.Error
	if errors.As(err, &e) && e.Status >= 400 && e.Status < 500 {
		return 2
	}
	return 1
}

func fail(stdout, stderr io.Writer, asJSON bool, err error, code int) int {
	if asJSON {
		problem := service.Problem{Code: "command_failed", Message: err.Error()}
		var e *service.Error
		if errors.As(err, &e) {
			problem.Code, problem.Message = e.Code, e.Message
		}
		if writeJSON(stdout, stderr, map[string]any{"error": problem}) != 0 {
			return 1
		}
	} else {
		fmt.Fprintln(stderr, err)
	}
	return code
}

func writeJSON(stdout, stderr io.Writer, value any) int {
	if err := json.NewEncoder(stdout).Encode(value); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
