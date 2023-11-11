package cmd

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/html/charset"
	"gopkg.in/yaml.v3"

	"github.com/itrs-group/cordial/pkg/geneos"
)

type Probe struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
	Port     int    `json:"port,omitempty"`
	Secure   bool   `json:"secure,omitempty"`
}

type Sampler struct {
	Type    string   `json:"type,omitempty"`
	Name    string   `json:"name"`
	Plugin  string   `json:"plugin,omitempty"`
	Column1 []string `json:"column1,omitempty"`
	Column2 []string `json:"remote-ports,omitempty"`
}

type Entity struct {
	Name       string            `json:"name"`
	Probe      Probe             `json:"probe"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Samplers   []Sampler         `json:"sampler,omitempty"`
}

func processInputFile(input io.Reader) (gateway string, entities []Entity, probeMap map[string]geneos.Probe, err error) {
	d := xml.NewDecoder(input)
	d.CharsetReader = charset.NewReaderLabel

	var g geneos.Gateway
	if err = d.Decode(&g); err != nil {
		line, column := d.InputPos()
		err = fmt.Errorf("%w: line: %d char: %d", err, line, column)
		return
	}

	if g.OperatingEnvironment == nil {
		log.Fatal().Msg("no operating environment found")
	}

	gateway = g.OperatingEnvironment.GatewayName

	probeMap = geneos.UnrollProbes(g.Probes)
	types := geneos.UnrollTypes(g.Types)
	procdesc := geneos.UnrollProcessDescriptors(g.ProcessDescriptors)
	entityMap := geneos.UnrollEntities(g.ManagedEntities, types)
	samplers := geneos.UnrollSamplers(g.Samplers)
	// rules := geneos.UnrollRules(g.Rules)
	// printStructJSON(os.Stderr, rules)

	// var Entities = []Entity{}

	var entnames []string
	for n := range entityMap {
		entnames = append(entnames, n)
	}
	sort.Strings(entnames)

	for _, entname := range entnames {
		entity := entityMap[entname]
		ent := Entity{}
		var probename string
		if entity.Probe != nil {
			probename = entity.Probe.Name
		} else if entity.FloatingProbe != nil {
			probename = entity.FloatingProbe.Name
		} else if entity.VirtualProbe != nil {
			probename = entity.VirtualProbe.Name
		} else {
			log.Error().Msgf("no probe set for %q\n", entity.Name)
			continue
		}

		probe, ok := probeMap[probename]
		if !ok {
			log.Error().Msgf("probe %q not found\n", probename)
		}
		var hostname string
		var port int
		var secure bool
		switch probe.Type {
		case geneos.ProbeTypeProbe:
			probename = probe.Name
			hostname = probe.Hostname
			port = probe.Port
			if probe.Secure != nil {
				secure = *probe.Secure
			}
		case geneos.ProbeTypeFloating:
			probename = probe.Name
		case geneos.ProbeTypeVirtual:
			probename = probe.Name
		default:
			log.Error().Msgf("%q probe %q not found\n", entity.Name, probename)
		}

		ent.Name = entity.Name
		ent.Probe = Probe{
			Name:     probename,
			Hostname: hostname,
			Port:     port,
			Secure:   secure,
		}

		ent.Samplers = []Sampler{}

		var names []string
		for s := range entity.ResolvedSamplers {
			names = append(names, s)
		}
		sort.Strings(names)

		for _, s := range names {
			// s := entity.ResolvedSamplers[sampname]
			// for s := range entity.ResolvedSamplers {
			p := strings.SplitN(s, ":", 2)
			plugin := geneos.GetPlugin(samplers[p[1]].Plugin)
			if plugin == nil {
				plugin = ""
			}
			sampler := &Sampler{
				Name:   p[1],
				Type:   p[0],
				Plugin: fmt.Sprint(plugin),
			}
			pluginInfo(sampler, plugin, procdesc)
			ent.Samplers = append(ent.Samplers, *sampler)
		}

		ent.Attributes = map[string]string{}
		for _, a := range entity.Attributes {
			ent.Attributes[a.Name] = a.Value
		}

		entities = append(entities, ent)
	}

	return
}

func printStructJSON(out io.Writer, in any) {
	j := json.NewEncoder(out)
	j.SetEscapeHTML(false)
	j.SetIndent("", "    ")
	err := j.Encode(in)
	if err != nil {
		return
	}
}

func printStructYAML(out io.Writer, in any) {
	b, _ := yaml.Marshal(in)
	fmt.Fprintln(out, string(b))
}

func pluginName(in interface{}) (name string) {
	switch t := in.(type) {
	case map[string]interface{}:
		for n := range t {
			return n
		}
	default:
		log.Debug().Msgf("unknown type %T", t)
	}
	return ""
}

func pluginInfo(sampler *Sampler, in interface{}, procdesc map[string]geneos.ProcessDescriptor) {
	// log.Info().Msgf("looking at plugin: %T", in)
	switch plugin := in.(type) {
	case *geneos.FKMPlugin:
		// grab files
		sampler.Column1 = []string{}
		files := plugin.Files.Files
		for _, file := range files {
			src := file.Source
			if src.Filename != nil {
				sampler.Column1 = append(sampler.Column1, src.Filename.String())
			} else if src.Stream != nil {
				sampler.Column1 = append(sampler.Column1, "stream:"+src.Stream.String())
			} else if src.ProcessDescriptor != nil {
				sampler.Column1 = append(sampler.Column1, procdesc[src.ProcessDescriptor.Name].LogFile.String())
			} else if src.NTEventLog != "" {
				sampler.Column1 = append(sampler.Column1, "NTEventLog:"+src.NTEventLog)
			} else {
				log.Error().Msg("unsupported FKM source tye")
			}
		}

	case *geneos.FTMPlugin:
		sampler.Column1 = []string{}
		for _, file := range plugin.Files {
			sampler.Column1 = append(sampler.Column1, file.Path.String())
			if file.AdditionalPaths != nil {
				for _, a := range file.AdditionalPaths.Paths {
					sampler.Column1 = append(sampler.Column1, a.String())
				}
			}
		}

	case *geneos.StateTrackerPlugin:
		sampler.Column1 = []string{}
		for _, group := range plugin.TrackerGroup {
			for _, tracker := range group.Trackers {
				name := tracker.Name.String()
				if name == "" {
					name = tracker.DeprecatedName + " [*]"
				}
				sampler.Column1 = append(sampler.Column1, fmt.Sprintf("%s : %s : %s", group.Name, name, tracker.Filename.String()))
			}
		}

	case *geneos.ProcessesPlugin:
		sampler.Column1 = []string{}
		for _, process := range plugin.Processes {
			if process.Data != nil {
				sampler.Column1 = append(sampler.Column1, process.Data.Alias.String())
			} else if process.ProcessDescriptor != nil {
				sampler.Column1 = append(sampler.Column1, procdesc[process.ProcessDescriptor.Name].Alias.String())
			} else {
				// panic("here")
			}
		}

	case *geneos.ToolkitPlugin:
		sampler.Column1 = append(sampler.Column1, plugin.SamplerScript.String())

	case *geneos.GatewaySeverityCountPlugin:
		// nothing

	case *geneos.DiskPlugin:
		sampler.Column1 = []string{}
		if plugin.AutoDetect != nil && plugin.AutoDetect.String() == "true" {
			sampler.Column1 = append(sampler.Column1, "[AUTODETECT]")
		}
		if plugin.CheckNFSPartitions != nil && plugin.CheckNFSPartitions.String() == "false" {
			sampler.Column1 = append(sampler.Column1, "[NO_NFS]")
		}
		for _, partition := range plugin.Partitions {
			var p string
			if partition.Path == nil {
				continue
			}
			p = partition.Path.String()
			if partition.Alias != nil {
				p = fmt.Sprintf("%s (%s)", p, partition.Alias.String())
			}
			sampler.Column1 = append(sampler.Column1, p)
		}
		for _, exclude := range plugin.ExcludePartitions {
			x := exclude.Path
			if x == "" {
				x = exclude.FileSystem
			} else if exclude.FileSystem != "" {
				x = fmt.Sprintf("%s (%s)", x, exclude.FileSystem)
			}
			sampler.Column1 = append(sampler.Column1, "[x] "+x)
		}

	case *geneos.CPUPlugin:
		// no additional data

	case *geneos.HardwarePlugin:
		// no additional data

	case *geneos.DeviceIOPlugin:
		// no additional data

	case *geneos.NetworkPlugin:
		// nothing yet

	case *geneos.TopPlugin:
		// nothing

	case *geneos.WebMonPlugin:
		// nothing yet

	case *geneos.UNIXUsersPlugin:
		// no additional data

	case *geneos.XPingPlugin:
		for _, t := range plugin.TargetNodes {
			sampler.Column1 = append(sampler.Column1, t.String())
		}

	case *geneos.GatewaySQLPlugin:
		for _, d := range plugin.Views {
			sampler.Column1 = append(sampler.Column1, d.ViewName.String())
		}

	case *geneos.SQLToolkitPlugin:
		sampler.Column1 = []string{plugin.Connection.String()}
		for _, q := range plugin.Queries {
			sampler.Column2 = append(sampler.Column2, fmt.Sprintf("%q: [\n%.*s\n]", q.Name, 32000, strings.TrimSpace(q.SQL.String())))
		}

	case *geneos.ControlMPlugin:
		for _, d := range plugin.Dataviews {
			// sampler.Dataviews = append(sampler.Dataviews, d.Name)
			for _, p := range d.Parameters {
				sampler.Column1 = append(sampler.Column1, fmt.Sprintf("%s : %s : %s", d.Name, p.Parameter, p.Criteria))
			}
		}

	case *geneos.TCPLinksPlugin:
		for _, l := range plugin.LocalPorts {
			if p := strings.TrimPrefix(l.String(), ":"); p != "" {
				sampler.Column1 = append(sampler.Column1, p)
			}
		}
		for _, r := range plugin.RemotePorts {
			if p := strings.TrimPrefix(r.String(), ":"); p != "" {
				sampler.Column2 = append(sampler.Column2, p)
			}
		}

	// case *geneos.JMXServerPlugin:
	// 	log.Debug().Msg("jmx-server not yet supported")

	case *geneos.WinServicesPlugin:
		for _, s := range plugin.Services {
			var sn string
			if s.ServiceName != nil {
				if s.ServiceName.Basic != nil {
					sn = s.ServiceName.Basic.String()
				} else if s.ServiceName.Regex != nil {
					sn = s.ServiceName.Regex.String()
				}
				if s.ServiceName.BasicShowUnavailable {
					sn += " *"
				}
			}

			var sd string
			if s.ServiceDescription != nil {
				if s.ServiceDescription.Basic != nil {
					sd = s.ServiceDescription.Basic.String()
				} else if s.ServiceDescription.Regex != nil {
					sd = s.ServiceDescription.Regex.String()
				}
			}

			if sn != "" && sd != "" {
				sampler.Column1 = append(sampler.Column1, fmt.Sprintf("%s [%s]", sd, sn))
			} else if sn != "" {
				sampler.Column1 = append(sampler.Column1, sn)
			} else if sd != "" {
				sampler.Column1 = append(sampler.Column1, sd)
			}
		}

		if len(sampler.Column1) == 0 {
			sampler.Column1 = append(sampler.Column1, "[ALL]")
		}

	case string:
		if plugin != "" {
			log.Debug().Msgf("plugin %q not yet supported", plugin)
		}
	default:
		log.Debug().Msgf("unsupported plugin %q skipped", plugin)

	}

}
