package profiles

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/itrs-group/cordial/pkg/config"
	"github.com/itrs-group/cordial/tools/geneos/internal/geneos"
	"github.com/itrs-group/cordial/tools/geneos/internal/instance"
	"github.com/itrs-group/cordial/tools/geneos/internal/responses"
	"github.com/itrs-group/cordial/tools/geneos/internal/values"
)

type Gateway struct {
	Common      `mapstructure:",squash"`
	GatewayName string            `yaml:"gateway-name"`
	LicdHost    string            `yaml:"licd-host,omitempty"`
	LicdPort    int               `yaml:"licd-port,omitempty"`
	LicdSecure  *bool             `yaml:"licd-secure,omitempty"`
	Includes    values.Includes   `yaml:"includes,omitempty"`
	Variables   []values.Variable `yaml:"variables,omitempty"`
}

func applyGateways(pf *config.Config, gateways []Gateway) {
	if len(gateways) == 0 {
		return
	}

	ct := geneos.ParseComponent("gateway")

	// global lookup table, add values here as needed
	lookup := map[string]string{
		"hostname": geneos.LOCAL.String(),
	}

	for _, gateway := range gateways {
		name := config.Expand[string](pf, gateway.Name, config.LookupTable(lookup))

		log.Debug("gateway name", slog.String("name", name))
		if name == "" {
			panic("name empty")
		}
		vals, err := applyCommonParams(gateway.Common, values.Values{})
		if err != nil {
			// handle the error appropriately, e.g., log it or return
			continue
		}

		vals.Params = append(vals.Params, "gatewayname="+gateway.GatewayName)
		vals.Params = append(vals.Params, "licdhost="+gateway.LicdHost)
		if gateway.LicdPort != 0 {
			vals.Params = append(vals.Params, fmt.Sprintf("licdport=%d", gateway.LicdPort))
		}
		log.Debug("licdsecure", slog.Any("licdsecure", gateway.LicdSecure))
		if gateway.LicdSecure != nil {
			vals.Params = append(vals.Params, fmt.Sprintf("licdsecure=%v", *gateway.LicdSecure))
		}
		vals.Includes = gateway.Includes
		vals.Variables = gateway.Variables

		instances := instance.Instances(geneos.LOCAL, ct, instance.MatchNames(name))
		log.Debug("retrieved Gateway instances", slog.String("name", name), slog.Int("count", len(instances)))
		if len(instances) == 0 {
			// create here
			log.Debug("creating Gateway instance", slog.String("name", name))
			instance.Add(geneos.LOCAL, ct, name, 0, vals,
				instance.CertBundle(gateway.Common.CertBundle),
				instance.CertBundlePassword(gateway.Common.CertBundlePassword),
			)
		} else {
			// update the first / only instance
			i := instances[0]
			log.Debug("updating Gateway instance", slog.String("name", i.Name()))
			cf := i.Config()
			keyfile := config.Get[config.KeyFile](cf, "keyfile")
			// these may or may not be reset by config, remove
			config.Delete(i.Config(), "licdsecure")
			config.Delete(i.Config(), "licdport")

			// Reset existing Gateway configuration for this instance before applying new values
			config.Delete(cf, values.INCLUDES)
			config.Delete(cf, values.VARIABLES)
			config.Delete(cf, values.ENVIRONMENT)

			if ncf, err := values.Set(i, vals, keyfile); err == nil {
				i.SetConfig(ncf)
			}

			if gateway.CertBundle != "" {
				updated, err := instance.ImportCertificates(i, gateway.CertBundle, "", gateway.CertBundlePassword)
				if err != nil {
					i.Log().Error("failed to import certificates", slog.Any("error", err))
				}
				if updated {
					i.Log().Debug("ca-bundle updated")
				}
			}
			if resp := instance.Write(i); resp.Err != nil {
				i.Log().Error("write failed", slog.Any("error", resp.Err))
				return
			}
			// rebuild as if new
			i.Rebuild(true)
		}
		instance.Do(geneos.LOCAL, ct, []string{name}, func(i geneos.Instance, a ...any) (resp *responses.General) {
			resp = responses.New[responses.General](i)
			resp.Err = instance.Start(i)
			return
		}, vals).Report(os.Stdout, responses.IgnoreErr(geneos.ErrRunning))
	}
}
