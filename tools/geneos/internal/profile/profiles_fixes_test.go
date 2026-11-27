package profile_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/cmd"
	"github.com/itrs-group/cordial/tools/geneos/internal/component/gateway"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneostest"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/profile"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

// setProfiles sets the profiles configuration from yaml for the test.
func setProfiles(t *testing.T, yaml string) *config.Config {
	t.Helper()
	previousProfiles, previousCmdProfiles := profile.Profiles, cmd.Profiles
	t.Cleanup(func() { profile.Profiles, cmd.Profiles = previousProfiles, previousCmdProfiles })
	pf := config.New(config.WithDefaults([]byte(yaml), "yaml"))
	profile.Profiles, cmd.Profiles = pf, pf
	return pf
}

func TestListComponentsOrderTemplateAndOverrides(t *testing.T) {
	geneostest.NewRoot(t, "testhost")
	pf := setProfiles(t, `profiles:
  ordered:
    _short: ignored
    webserver:
      - name: ${hostname}
    licd:
      - name: second
      - name: first
        template: /custom/template.gotmpl
    gateway:
      name: single
    netprobe:
      - name: minimal:${hostname}
`)
	profile.SetExpandOptions(geneos.LOCAL, pf, nil, nil)

	got := profile.ListComponents(geneos.LOCAL, pf, "ordered")
	var names []string
	for _, pi := range got {
		names = append(names, pi.Component+"/"+pi.Name+"/"+pi.Package)
	}
	want := []string{
		"gateway/single/gateway",
		"licd/second/licd",
		"licd/first/licd",
		"netprobe/minimal:testhost/minimal",
		"webserver/testhost/webserver",
	}
	if !slices.Equal(names, want) {
		t.Fatalf("ListComponents() = %v, want %v", names, want)
	}

	// the entry's own settings are available, e.g. for a template override
	for _, pi := range got {
		template, ok := config.Lookup[string](pi.Config, "template")
		if want := pi.Name == "first"; ok != want || (ok && template != "/custom/template.gotmpl") {
			t.Errorf("%s template = %q, %t", pi.Name, template, ok)
		}
	}

	if got := profile.ListComponents(geneos.LOCAL, pf, "missing"); len(got) != 0 {
		t.Errorf("ListComponents() for a missing profile = %v", got)
	}
}

func TestPflagSetToMap(t *testing.T) {
	if got := profile.PflagSetToMap(nil, "insecure"); got == nil || len(got) != 0 {
		t.Errorf("PflagSetToMap(nil) = %#v, want an empty map", got)
	}

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.Bool("insecure", true, "")
	fs.Int("count", 3, "")
	fs.Int8("small", 4, "")
	fs.Uint8("usmall", 5, "")
	fs.Float64("ratio", 1.5, "")
	fs.String("name", "x", "")
	fs.StringSlice("list", []string{"a", "b"}, "")
	fs.Bool("debug", true, "")
	var gateways values.Gateways
	fs.Var(&gateways, "gateway", "")
	_ = fs.Set("gateway", "a:7039")
	_ = fs.Set("gateway", "b:7100")

	got := profile.PflagSetToMap(fs)
	want := map[string]any{
		"insecure": true,
		"count":    3,
		"small":    4,
		"usmall":   uint64(5),
		"ratio":    1.5,
		"name":     "x",
		"list":     "a,b",
		"gateway":  "a:7039,b:7100",
	}
	if len(got) != len(want) {
		t.Errorf("PflagSetToMap() = %#v, want %#v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("PflagSetToMap()[%q] = %#v, want %#v", k, got[k], v)
		}
	}

	if got := profile.PflagSetToMap(fs, "insecure", "g*"); len(got) != 2 {
		t.Errorf("PflagSetToMap() with a filter = %#v", got)
	}
}

func TestExpandCommaList(t *testing.T) {
	pf := config.New()
	profile.SetExpandOptions(geneos.LOCAL, pf, map[string]any{"gateway": "a:7039,b:7100", "empty": ""}, nil)

	for _, test := range []struct {
		in   []string
		want []string
	}{
		{nil, nil},
		{[]string{""}, nil},
		{[]string{"${option:gateway}"}, []string{"a:7039", "b:7100"}},
		{[]string{"x:1", "${option:gateway}", " y:2 "}, []string{"x:1", "a:7039", "b:7100", "y:2"}},
		{[]string{"${option:empty?\"a\":\"\"}", ""}, nil},
	} {
		if got := profile.ExpandCommaList(pf, test.in); !slices.Equal(got, test.want) {
			t.Errorf("ExpandCommaList(%q) = %q, want %q", test.in, got, test.want)
		}
	}
}

func TestPrefixFunctionsUseNonStringValuesAndHyphenatedKeys(t *testing.T) {
	ci := map[string]any{
		"port":         7039,
		"enabled":      true,
		"java-options": "-Xmx1g",
		"name":         "my gateway",
		"empty":        "",
		"list":         []any{"a"},
	}
	for _, test := range []struct {
		expr string
		want string
	}{
		{"select:port:none", "7039"},
		{"select:enabled:none", "true"},
		{"select:java-options:none", "-Xmx1g"},
		{"select:empty:name:none", "my gateway"},
		{"select:list:none", "none"},
		{"select:name-port:none", "my gateway-7039"},
	} {
		got, err := profile.SelectPrefix(ci, test.expr, false)
		if err != nil || got != test.want {
			t.Errorf("SelectPrefix(%q) = %q, %v, want %q", test.expr, got, err, test.want)
		}
	}

	if got, err := profile.ReplacePrefix(ci, "replace:port:/70/80/", false); err != nil || got != "8039" {
		t.Errorf("ReplacePrefix() on a number = %q, %v", got, err)
	}
	if got, err := profile.ReplacePrefix(ci, "replace:java-options:/-X/-Y/", false); err != nil || got != "-Ymx1g" {
		t.Errorf("ReplacePrefix() on a hyphenated key = %q, %v", got, err)
	}
	for _, bad := range []string{"replace:", "replace:name:", "replace:name:/a/b"} {
		if _, err := profile.ReplacePrefix(ci, bad, false); err == nil {
			t.Errorf("ReplacePrefix(%q) did not fail", bad)
		}
	}
}

func TestPrefixFunctionsExpandNestedPrefixes(t *testing.T) {
	pf := config.New()
	config.Set(pf, "home", "/opt/geneos")
	config.Set(pf, "label", "${option:level} gateway")
	config.Set(pf, "alias", "${select:label:none}")
	config.Set(pf, "loop", "${select:loop:none}")
	profile.SetExpandOptions(geneos.LOCAL, pf, map[string]any{"level": "prod"}, nil)

	if got := config.Expand[string](pf, "${select:label:none}"); got != "prod gateway" {
		t.Errorf("select of a value with an option = %q", got)
	}
	if got := config.Expand[string](pf, "${select:alias:none}"); got != "prod gateway" {
		t.Errorf("select of a value with a nested select = %q", got)
	}
	if got := config.Expand[string](pf, "${replace:label:/prod/test/}"); got != "test gateway" {
		t.Errorf("replace of a value with an option = %q", got)
	}
	// a value that refers to itself must terminate
	_ = config.Expand[string](pf, "${select:loop:none}")
}

func TestJoinPathUsesHostAndKeepsOptionalSegments(t *testing.T) {
	geneostest.NewRoot(t, "")
	ci := map[string]any{"home": "/opt/gw", "logfile": "gateway.log"}
	got, err := profile.JoinPathPrefix(ci, "joinpath:home:logdir:logfile", false, geneos.LOCAL)
	if err != nil || got != "/opt/gw/gateway.log" {
		t.Errorf("JoinPathPrefix() = %q, %v", got, err)
	}

	for logdir, want := range map[string]string{
		"logs":     "/opt/gw/logs/gateway.log",
		"/var/log": "/var/log/gateway.log",
	} {
		ci["logdir"] = logdir
		got, err := profile.JoinPathPrefix(ci, "joinpath:home:logdir:logfile", false, geneos.LOCAL)
		if err != nil || got != want {
			t.Errorf("logdir %q: JoinPathPrefix() = %q, %v, want %q", logdir, got, err, want)
		}
	}
}

func TestSetExpandOptionsCoercesLegacyBooleans(t *testing.T) {
	for _, test := range []struct {
		value any
		want  string
	}{
		{true, "8080"},
		{false, "8443"},
		{"1", "8080"},
		{"0", "8443"},
		{"true", "8080"},
		{"false", "8443"},
	} {
		t.Run(fmt.Sprint(test.value), func(t *testing.T) {
			pf := config.New()
			flags := map[string]any{"insecure": test.value, "debug": true}
			profile.SetExpandOptions(geneos.LOCAL, pf, flags, nil)
			if got := config.Expand[string](pf, "${option:insecure?8080:8443}"); got != test.want {
				t.Errorf("expansion = %q, want %q", got, test.want)
			}
			if flags["insecure"] != test.value || flags["debug"] != true {
				t.Errorf("SetExpandOptions changed the caller's flags: %v", flags)
			}
		})
	}
}

func TestApplyUnknownComponentIsAnError(t *testing.T) {
	geneostest.NewRoot(t, "")
	pf := setProfiles(t, "profiles:\n  bad:\n    notacomponent:\n      - name: x\n")
	profile.SetExpandOptions(geneos.LOCAL, pf, nil, nil)
	if err := profile.Apply(geneos.LOCAL, pf, "bad", false); err == nil {
		t.Fatal("Apply() of a profile with an unknown component succeeded")
	}
}

func TestApplyInstanceRejectsInvalidPort(t *testing.T) {
	geneostest.NewRoot(t, "", "licd")
	ct := geneos.ParseComponent("licd")
	pf := config.New()
	for _, port := range []string{"70000", "-1", "abc", "8080x"} {
		t.Run(port, func(t *testing.T) {
			common := profile.Common{Name: "port-" + strings.TrimPrefix(port, "-"), Params: []string{"port=" + port}}
			vals := profile.ApplyCommonParams(pf, "ports", &common, nil)
			err := profile.ApplyInstance(geneos.LOCAL, pf, ct, common.Name, common, vals, nil, false)
			if err == nil || !strings.Contains(err.Error(), "invalid port") {
				t.Fatalf("ApplyInstance() = %v, want an invalid port error", err)
			}
			if existing := instance.Instances(geneos.LOCAL, ct, instance.MatchNames(common.Name)); len(existing) != 0 {
				t.Errorf("an instance was created with an invalid port: %v", existing)
			}
		})
	}
}

func TestApplyCommonProfileClearsRemovedEnvironment(t *testing.T) {
	geneostest.NewRoot(t, "", "licd")
	ct := geneos.ParseComponent("licd")
	pf := setProfiles(t, "profiles:\n  envs:\n    licd:\n      - name: env-member\n        env:\n          - A=1\n")
	profile.SetExpandOptions(geneos.LOCAL, pf, nil, nil)
	if err := profile.Apply(geneos.LOCAL, pf, "envs", false); err != nil {
		t.Fatal(err)
	}
	member, err := instance.GetWithHost(geneos.LOCAL, ct, "env-member")
	if err != nil {
		t.Fatal(err)
	}
	config.Set(member.Config(), values.ENVIRONMENT, []string{"A=1", "OLD=1"})
	if resp := instance.Write(member, instance.NoRebuild()); resp.Err != nil {
		t.Fatal(resp.Err)
	}
	if err := profile.Apply(geneos.LOCAL, pf, "envs", false); err != nil {
		t.Fatal(err)
	}
	member, err = instance.GetWithHost(geneos.LOCAL, ct, "env-member")
	if err != nil {
		t.Fatal(err)
	}
	if got := config.Get[[]string](member.Config(), values.ENVIRONMENT); !slices.Equal(got, []string{"A=1"}) {
		t.Errorf("env = %v, want only the value from the profile", got)
	}
}

func TestDeployProfilePortNeedsSingleComponent(t *testing.T) {
	geneostest.NewRoot(t, "", "licd", "gateway")
	pf := setProfiles(t, `profiles:
  two:
    licd:
      - name: one
    gateway:
      - name: two
  one:
    licd:
      - name: only
`)
	profile.SetExpandOptions(geneos.LOCAL, pf, nil, nil)

	err := profile.DeployProfile(geneos.LOCAL, "two", profile.WithPort(7100), profile.WithInsecure(true),
		profile.WithFlags(pflag.NewFlagSet("test", pflag.ContinueOnError)))
	if !errors.Is(err, geneos.ErrInvalidArgs) {
		t.Fatalf("DeployProfile() with a port and two components = %v, want an invalid argument error", err)
	}
	// nothing is created before the error
	if all := instance.Instances(geneos.LOCAL, nil); len(all) != 0 {
		t.Errorf("instances created before the error: %v", all)
	}

	port := uint16(7141)
	if err := profile.DeployProfile(geneos.LOCAL, "one", profile.WithPort(port), profile.WithInsecure(true),
		profile.WithFlags(pflag.NewFlagSet("test", pflag.ContinueOnError))); err != nil {
		t.Fatal(err)
	}
	member, err := instance.GetWithHost(geneos.LOCAL, geneos.ParseComponent("licd"), "only")
	if err != nil {
		t.Fatal(err)
	}
	if got := config.Get[uint16](member.Config(), "port"); got != port {
		t.Errorf("port = %d, want %d", got, port)
	}
}

func TestDeployProfileReportsInstallPrompt(t *testing.T) {
	// no package is installed, so one is needed, but with a username and
	// no terminal for the password there is no way to download it
	geneostest.NewRoot(t, "")
	pf := setProfiles(t, "profiles:\n  needs-install:\n    licd:\n      - name: member\n")
	profile.SetExpandOptions(geneos.LOCAL, pf, nil, nil)

	err := profile.DeployProfile(geneos.LOCAL, "needs-install", profile.WithUsername("someone@example.com"),
		profile.WithInsecure(true), profile.WithFlags(pflag.NewFlagSet("test", pflag.ContinueOnError)))
	if !errors.Is(err, config.ErrNotInteractive) {
		t.Fatalf("DeployProfile() = %v, want a not interactive error", err)
	}
	if all := instance.Instances(geneos.LOCAL, nil); len(all) != 0 {
		t.Errorf("instances created after the error: %v", all)
	}
}

func TestDeployProfileUsesTemplateOverride(t *testing.T) {
	root := geneostest.NewRoot(t, "", "gateway")
	custom := filepath.Join(root, "custom.gotmpl")
	if err := os.WriteFile(custom, []byte("custom template"), 0o600); err != nil {
		t.Fatal(err)
	}
	pf := setProfiles(t, fmt.Sprintf("profiles:\n  tmpl:\n    gateway:\n      - name: tgw\n        template: %s\n", custom))
	profile.SetExpandOptions(geneos.LOCAL, pf, nil, nil)

	if err := profile.DeployProfile(geneos.LOCAL, "tmpl", profile.WithInsecure(true),
		profile.WithFlags(pflag.NewFlagSet("test", pflag.ContinueOnError))); err != nil {
		t.Fatal(err)
	}
	for _, template := range gateway.Component.Templates {
		got, err := os.ReadFile(filepath.Join(root, "gateway", "templates", template.Filename))
		if err != nil || string(got) != "custom template" {
			t.Errorf("template %s = %q, %v, want the profile template", template.Filename, got, err)
		}
	}
}

func TestEmbeddedHealthGatewayIncludePriorities(t *testing.T) {
	pf := config.New(config.WithDefaults(profile.EmbeddedDefaults(), "yaml"))
	var gateways []gateway.GatewayProfile
	if err := pf.UnmarshalKey(pf.Join("profiles", "health-gateway", "gateways"), &gateways, config.NoExpand()); err != nil {
		t.Fatal(err)
	}
	if len(gateways) != 1 {
		t.Fatalf("got %d gateways, want 1", len(gateways))
	}
	var priorities []string
	for priority := range gateways[0].Includes {
		priorities = append(priorities, priority)
	}
	slices.Sort(priorities)
	// decimal priorities, and not leading zero octal numbers
	if want := []string{"10000", "1010", "1020", "11000", "500", "510", "520"}; !slices.Equal(priorities, want) {
		t.Errorf("include priorities = %v, want %v", priorities, want)
	}
}

func TestGatewayProfileAppliesNameLicenceAndInstanceVariables(t *testing.T) {
	geneostest.NewRoot(t, "testhost", "gateway")
	pf := setProfiles(t, `profiles:
  gw:
    gateway:
      - name: ${hostname} Health
        gateway-name: Health for ${hostname}
        licd-host: licence.example.com
        licd-port: 7141
        licd-secure: false
        variables:
          - name: logFilePath
            type: string
            value: ${config:home}/${config:logfile}
          - name: host
            value: ${hostname}
`)
	profile.SetExpandOptions(geneos.LOCAL, pf, nil, nil)
	if err := profile.Apply(geneos.LOCAL, pf, "gw", false); err != nil {
		t.Fatal(err)
	}
	member, err := instance.GetWithHost(geneos.LOCAL, geneos.ParseComponent("gateway"), "testhost Health")
	if err != nil {
		t.Fatal(err)
	}
	cf := member.Config()
	for key, want := range map[string]string{
		"gatewayname": "Health for testhost",
		"licdhost":    "licence.example.com",
		"licdport":    "7141",
		"licdsecure":  "false",
	} {
		if got := config.Get[string](cf, key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}

	vars, _ := values.NormaliseVars(config.Get[[]any](cf, values.VARIABLES, config.NoExpand()))
	byName := map[string]string{}
	for _, v := range vars {
		byName[v.Name] = v.Value
	}
	if want := filepath.Join(member.Home(), "gateway.log"); byName["logFilePath"] != want {
		t.Errorf("logFilePath = %q, want %q", byName["logFilePath"], want)
	}
	if byName["host"] != "testhost" {
		t.Errorf("host = %q, want testhost", byName["host"])
	}
}
