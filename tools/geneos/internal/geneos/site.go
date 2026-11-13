package geneos

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/itrs-group/cordial"
	"github.com/itrs-group/cordial/pkg/config"
)

// SiteName returns the site name used in include paths.
func SiteName() string {
	site := config.Get[string](config.Global(), "site")
	if site == "" {
		return "local"
	}
	return strings.ReplaceAll(site, " ", "_")
}

// ValidateSite rejects unsafe path components and reserved Gateway directory names.
func ValidateSite(site string) error {
	if site == "" {
		return nil
	}
	name := strings.ReplaceAll(site, " ", "_")
	if name == "." || name == ".." || strings.ContainsAny(name, `/\:*?"<>|`) || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return fmt.Errorf("invalid site name %q: must be a single safe path component", site)
	}
	switch strings.ToLower(name) {
	case "config", "include", "includes", "gateways", "shared", "gateway_shared", "gateway_config", "templates":
		return fmt.Errorf("invalid site name %q: conflicts with a Gateway directory", site)
	}
	return nil
}

// SetInitialSite persists the initial site without replacing an existing one.
func SetInitialSite(site string) error {
	if err := ValidateSite(site); err != nil {
		return err
	}
	if site == "" {
		return fmt.Errorf("site name must not be empty")
	}
	current := config.Get[string](config.Global(), "site")
	if current != "" {
		if current != site {
			return fmt.Errorf("site is already set to %q; change it explicitly with geneos config set", current)
		}
		return nil
	}
	cf, err := config.Read(cordial.ExecutableName(), config.SkipSystemDir(), config.SkipWorkingDir())
	if err != nil {
		return fmt.Errorf("read configuration for site: %w", err)
	}
	config.Set(cf, "site", site)
	if err := cf.Write(cordial.ExecutableName()); err != nil {
		return fmt.Errorf("save site: %w", err)
	}
	config.Set(config.Global(), "site", site)
	return nil
}
