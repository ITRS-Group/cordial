package geneos

type Options struct {
	override     string
	local        bool
	nosave       bool
	overwrite    bool
	restart      bool
	basename     string
	homedir      string
	version      string
	username     string
	password     string
	platform_id  string
	downloadbase string
	downloadtype string
	filename     string
}

type GeneosOptions func(*Options)

func EvalOptions(options ...GeneosOptions) (d *Options) {
	// defaults
	d = &Options{
		downloadbase: "releases",
		downloadtype: "resources",
	}
	for _, opt := range options {
		opt(d)
	}
	return
}

func NoSave(n bool) GeneosOptions {
	return func(d *Options) { d.nosave = n }
}

func LocalOnly(l bool) GeneosOptions {
	return func(d *Options) { d.local = l }
}

func Force(o bool) GeneosOptions {
	return func(d *Options) { d.overwrite = o }
}

func OverrideVersion(s string) GeneosOptions {
	return func(d *Options) { d.override = s }
}

func Restart(r bool) GeneosOptions {
	return func(d *Options) { d.restart = r }
}

func (d *Options) Restart() bool {
	return d.restart
}

func Version(v string) GeneosOptions {
	return func(d *Options) { d.version = v }
}

func Basename(b string) GeneosOptions {
	return func(d *Options) { d.basename = b }
}

func Homedir(h string) GeneosOptions {
	return func(d *Options) { d.homedir = h }
}

func Username(u string) GeneosOptions {
	return func(d *Options) { d.username = u }
}

func Password(p string) GeneosOptions {
	return func(d *Options) { d.password = p }
}

func PlatformID(id string) GeneosOptions {
	return func(d *Options) { d.platform_id = id }
}

func UseNexus() GeneosOptions {
	return func(d *Options) { d.downloadtype = "nexus" }
}

func UseSnapshots() GeneosOptions {
	return func(d *Options) { d.downloadbase = "snapshots" }
}

func Filename(f string) GeneosOptions {
	return func(d *Options) { d.filename = f }
}
