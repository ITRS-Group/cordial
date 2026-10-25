package profile

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"

	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

func DeployProfile(h *geneos.Host, name string, options ...ProfileDeployOption) (err error) {
	opts := evaluateProfileDeployOptions(options...)

	// discover which components are in the profile
	pf := Profiles

	instances := ListComponents(h, pf, name)
	log.Debug("components in profile", slog.String("host", h.String()), slog.String("hostname", h.Hostname()), slog.String("profile", name), slog.Any("components", instances))

	installed := []geneos.Instance{}
	for _, pi := range instances {
		ct := geneos.ParseComponent(pi.Component)
		if ct == nil {
			log.Debug("unknown component type", slog.String("component", pi.Component))
			continue
		}

		pkgct := geneos.ParseComponent(pi.Package)

		if pkgct == nil {
			log.Debug("unknown package component type", slog.String("package", pi.Package))
			continue
		}

		// create required component directories, for pkg type, speculatively
		if err = pkgct.MakeDirs(h); err != nil {
			return err
		}

		// deploy templates if component requires them, do not overwrite
		// existing
		//
		// templates are based on real component type (e.g. san and not fa2)
		if len(ct.Templates) != 0 {
			templateDir := h.PathTo(ct, "templates")
			h.MkdirAll(templateDir, 0775)

			// TODO: loop and single config makes no sense
			for _, t := range ct.Templates {
				tmpl := t.Content
				output := path.Join(templateDir, t.Filename)
				if _, err := h.Stat(output); err == nil {
					continue
				}
				if t, ok := config.Lookup[string](pi.Config, "template"); ok {
					if tmpl, err = geneos.ReadAll(t); err != nil {
						continue
					}
				}
				if err = h.WriteFile(output, tmpl, 0664); err != nil {
					continue
				}
				fmt.Printf("%s template %q written to %s\n", ct, t.Filename, templateDir)
			}
		}

		// Package installation

		version, _ := geneos.CurrentVersion(h, pkgct, opts.base)
		log.Debug("version", slog.String("component", pkgct.String()), slog.String("version", version))
		if version == "unknown" || (opts.version != "latest" && opts.version != version) {
			if !opts.local && opts.username != "" && opts.password == nil {
				opts.password, err = config.ReadPasswordInput(false, 0)
				if err == config.ErrNotInteractive {
					err = fmt.Errorf("%w and password required", err)
					continue
				}
				// ok to defer clear here as the options are used before returning to caller
				defer clear(opts.password)
			}

			if opts.snapshot {
				opts.nexus = true
			}
			options := []geneos.GeneosOption{
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
				options = append(options,
					geneos.Source(opts.archive),
				)
			}

			log.Debug("installing", slog.String("host", h.String()), slog.String("package", pkgct.String()))

			if err = geneos.Install(h, pkgct, options...); err != nil {
				if errors.Is(err, fs.ErrExist) {
					err = nil
				} else {
					continue
				}
			}

			// TODO: pull in values from profiles per instance
			if err := ct.MakeDirs(h); err != nil {
				continue
			}
			if ct.Initialise != nil {
				if err := ct.Initialise(h, ct, false); err != nil {
					log.Error("initialising component", slog.String("component", ct.Name), slog.Any("error", err))
				}
			}
			// clear any command line parameters, they should only come from the config
			extras := opts.vals
			extras.Params = []string{}
			extras.SecureParams = values.SecureValues{}

			i, err := instance.Add(h, ct, pi.Name,
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
			if err != nil {
				return err
			}
			installed = append(installed, i)
		}
	}

	// finally, install the profile instance
	_, err = instance.Add(h, geneos.ParseComponent("profile"), name,
		instance.WithPort(opts.port),
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
	)

	if err != nil {
		log.Error("failed to add profile instance", slog.Any("error", err))
		return err
	}

	if opts.followLogs {
		instance.FollowInstanceLogs(installed, instance.WithStderr(true))
	}

	return nil
}
