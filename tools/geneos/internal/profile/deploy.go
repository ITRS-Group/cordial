package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"sync"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

// initialised records the components that have been initialised for
// each host, as a component's Initialise function is only required once
// for each host.
var initialised sync.Map

// DeployProfile deploys the profile `name` on host h. For each component
// in the profile, in sorted order of component type, the required
// directories and templates are created, the package is installed if
// the required version is not already, and the instance is created if
// it does not already exist. Instances are always created in the same
// way, whether or not the package was installed by this call. Then the
// profile instance itself is created, or updated if it already exists,
// which applies the profile to all of the instances.
//
// The port option can only be used with a profile that has a single
// component. A profile that is not defined is an error. Every
// component is tried, but any error installing or creating them is
// returned together and then the profile instance is not created.
func DeployProfile(h *geneos.Host, name string, options ...ProfileDeployOption) (err error) {
	opts := evaluateProfileDeployOptions(options...)

	// discover which components are in the profile
	pf := Profiles
	if pf == nil {
		return errors.New("no profiles configuration loaded")
	}

	// the failures of the individual components are returned together
	// after all of them have been tried
	var errs []error

	instances := ListComponents(h, pf, name)
	log.Debug("components in profile", slog.String("host", h.String()), slog.String("hostname", h.Hostname()), slog.String("profile", name), slog.Any("components", instances))
	if len(instances) == 0 {
		return fmt.Errorf("%w: profile %q is not defined or has no components", geneos.ErrNotExist, name)
	}

	if opts.port != 0 && len(instances) != 1 {
		return fmt.Errorf("%w: a port can only be set for a profile with one component, profile %q has %d", geneos.ErrInvalidArgs, name, len(instances))
	}

	// only clear the password if it was read here
	defer func() {
		if opts.promptedPassword {
			clear(opts.password)
		}
	}()

	for _, pi := range instances {
		if cerr := deployComponent(h, pf, opts, pi); cerr != nil {
			errs = append(errs, fmt.Errorf("%s %s: %w", pi.Component, pi.Name, cerr))
		}
	}

	// the profile instance is only created if all of the components are
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	// finally, install the profile instance. the port is for the
	// component instance and not for the profile
	profileOptions := []instance.AddOption{
		instance.WithValues(opts.vals),
		instance.Base(opts.base),
		instance.Insecure(opts.insecure),
		instance.WithCertBundle(opts.certBundle),
		instance.WithCertBundlePassword(opts.certBundlePassword),
		instance.WithKeyfile(opts.keyfile),
		instance.KeyfileCRC(opts.keyfileCRC),
		instance.WithImports(opts.importFiles),
		instance.StartAfterAdd(opts.startAfterAdd),
		instance.WithFlags(opts.flags),
		instance.WithProfiles(pf),
	}

	pct := geneos.ParseComponent("profile")
	if existing := instance.Instances(h, pct, instance.MatchNames(name)); len(existing) > 0 {
		// the profile instance may be left over from an earlier deploy
		// that failed part way, so re-apply it to update the member
		// instances, which are added above without profile values
		err = reapplyProfile(existing[0], profileOptions)
	} else {
		_, err = instance.Add(h, pct, name, profileOptions...)
	}

	if err != nil {
		log.Error("failed to add profile instance", slog.Any("error", err))
		return err
	}

	// applying the profile may have changed the port of the only
	// component from the one requested, so set it again
	if opts.port != 0 {
		if err = setMemberPort(h, instances[0], opts.port); err != nil {
			return err
		}
	}

	if opts.followLogs {
		members := instance.Instances(h, nil, instance.MatchParameters(config.Join("config", "profile")+"="+name))
		instance.FollowInstanceLogs(members, instance.WithStderr(true))
	}

	return nil
}

// deployComponent prepares for and creates the instance for one
// component in a profile, if it does not exist.
func deployComponent(h *geneos.Host, pf *config.Config, opts *profileDeployOptions, pi Instance) (err error) {
	ct := geneos.ParseComponent(pi.Component)
	if ct == nil {
		return fmt.Errorf("%w: unknown component type %q in profile", geneos.ErrInvalidArgs, pi.Component)
	}

	pkgct := geneos.ParseComponent(pi.Package)
	if pkgct == nil {
		return fmt.Errorf("%w: unknown package component type %q in profile", geneos.ErrInvalidArgs, pi.Package)
	}

	// create required component directories, for pkg type, speculatively
	if err = pkgct.MakeDirs(h); err != nil {
		return err
	}

	// deploy templates if component requires them, do not overwrite
	// existing
	//
	// templates are based on real component type (e.g. san and not fa2)
	if err = deployTemplates(h, ct, pi); err != nil {
		return err
	}

	if err = installPackage(h, opts, pkgct); err != nil {
		return err
	}

	if err = ct.MakeDirs(h); err != nil {
		return err
	}
	if err = initialiseComponent(h, ct); err != nil {
		return err
	}

	if len(instance.Instances(h, ct, instance.MatchNames(pi.Name))) > 0 {
		log.Debug("instance already exists", slog.String("component", ct.String()), slog.String("name", pi.Name))
		return nil
	}

	// TODO: pull in values from profiles per instance
	//
	// clear any command line parameters, they should only come from the config
	extras := opts.vals
	extras.Params = []string{}
	extras.SecureParams = values.SecureValues{}

	_, err = instance.Add(h, ct, pi.Name,
		instance.WithPort(opts.port),
		instance.WithValues(extras),
		instance.WithTemplate(opts.template),
		instance.Base(opts.base),
		instance.Insecure(opts.insecure),
		instance.WithCertBundle(opts.certBundle),
		instance.WithCertBundlePassword(opts.certBundlePassword),
		instance.WithKeyfile(opts.keyfile),
		instance.KeyfileCRC(opts.keyfileCRC),
		instance.WithImports(opts.importFiles),
		instance.WithFlags(opts.flags),
		instance.WithProfiles(pf),
	)
	// an existing member is updated by the profile, not an error
	if errors.Is(err, geneos.ErrExists) {
		return nil
	}
	return err
}

// deployTemplates writes the templates for the component, unless they
// already exist. A `template` in the profile entry overrides the
// built-in template.
func deployTemplates(h *geneos.Host, ct *geneos.Component, pi Instance) (err error) {
	if len(ct.Templates) == 0 {
		return nil
	}

	templateDir := h.PathTo(ct, "templates")
	if err = h.MkdirAll(templateDir, 0775); err != nil {
		return err
	}

	// TODO: loop and single config makes no sense
	for _, t := range ct.Templates {
		output := h.Join(templateDir, t.Filename)
		if _, err := h.Stat(output); err == nil {
			continue
		}
		content := t.Content
		if source, ok := config.Lookup[string](pi.Config, "template"); ok {
			if content, err = geneos.ReadAll(source); err != nil {
				return fmt.Errorf("read profile template %q: %w", source, err)
			}
		}
		if err = h.WriteFile(output, content, 0664); err != nil {
			return fmt.Errorf("write template %q: %w", output, err)
		}
		fmt.Printf("%s template %q written to %s\n", ct, t.Filename, templateDir)
	}
	return nil
}

// installPackage installs the package for pkgct if the required version
// is not already installed. A package that already exists is not an
// error.
func installPackage(h *geneos.Host, opts *profileDeployOptions, pkgct *geneos.Component) (err error) {
	version, _ := geneos.CurrentVersion(h, pkgct, opts.base)
	log.Debug("version", slog.String("component", pkgct.String()), slog.String("version", version))
	if version != "unknown" && (opts.version == "latest" || opts.version == version) {
		return nil
	}

	if !opts.local && opts.username != "" && opts.password == nil {
		opts.password, err = config.ReadPasswordInput(false, 0)
		if err != nil {
			if errors.Is(err, config.ErrNotInteractive) {
				return fmt.Errorf("%w and password required", err)
			}
			return err
		}
		opts.promptedPassword = true
	}

	if opts.snapshot {
		opts.nexus = true
	}
	installOptions := []geneos.GeneosOption{
		geneos.Version(opts.version),
		geneos.Basename(opts.base),
		geneos.UseRoot(config.Get[string](h.Config, cordial.ExecutableName())),
		geneos.LocalOnly(opts.local),
		geneos.NoSave(opts.noSave || opts.local),
		geneos.OverrideVersion(opts.override),
		geneos.Password(opts.password),
		geneos.Username(opts.username),
		geneos.Headers(opts.vals.Headers...),
		geneos.UseNexusSnapshots(opts.snapshot),
		geneos.UseNexus(opts.nexus),
	}
	if opts.archive != "" {
		installOptions = append(installOptions, geneos.Source(opts.archive))
	}

	log.Debug("installing", slog.String("host", h.String()), slog.String("package", pkgct.String()))

	if err = geneos.Install(h, pkgct, installOptions...); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return nil
		}
		if opts.archive != "" {
			return fmt.Errorf("install %s from %q: %w", pkgct, opts.archive, err)
		}
		return fmt.Errorf("install %s: %w", pkgct, err)
	}
	return nil
}

// initialiseComponent runs the component's Initialise function, once
// for each host.
func initialiseComponent(h *geneos.Host, ct *geneos.Component) error {
	if ct.Initialise == nil {
		return nil
	}
	if _, done := initialised.LoadOrStore(h.String()+":"+ct.Name, struct{}{}); done {
		return nil
	}
	if err := ct.Initialise(h, ct, false); err != nil {
		// allow another attempt
		initialised.Delete(h.String() + ":" + ct.Name)
		return fmt.Errorf("initialising component %s: %w", ct.Name, err)
	}
	return nil
}

// setMemberPort sets the port of the instance for the profile entry pi
// to port, if it is not already.
func setMemberPort(h *geneos.Host, pi Instance, port uint16) error {
	ct := geneos.ParseComponent(pi.Component)
	members := instance.Instances(h, ct, instance.MatchNames(pi.Name))
	if len(members) == 0 {
		return nil
	}
	member := members[0]
	if config.Get[uint16](member.Config(), "port") == port {
		return nil
	}
	config.Set(member.Config(), "port", port)
	if resp := instance.Write(member, instance.NoReload()); resp.Err != nil {
		return fmt.Errorf("set port %d for %s: %w", port, member, resp.Err)
	}
	return nil
}

// reapplyProfile updates the saved options of an existing profile
// instance and applies the profile again, using the profile component's
// Add method, which does both.
func reapplyProfile(i geneos.Instance, options []instance.AddOption) (err error) {
	if config.Get[bool](i.Config(), "protected") {
		return fmt.Errorf("%s is protected, skipping", i)
	}

	addOptions := make([]any, len(options))
	for n, o := range options {
		addOptions[n] = o
	}
	if err = i.Add(addOptions...); err != nil {
		return err
	}
	if resp := instance.Write(i, instance.NoRebuild()); resp.Err != nil {
		return resp.Err
	}
	fmt.Printf("%s updated\n", i)
	i.AuditEvent("action/update", nil)

	if instance.EvalAddOptions(addOptions...).Start {
		if err = instance.Start(i); errors.Is(err, os.ErrProcessDone) {
			err = nil
		}
	}
	return
}
