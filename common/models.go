package common

type WatcherConfig struct {
	Path   string
	Watch  []string
	Ignore []string
}

// Docker flags
type ComposeFlags struct {
	WC      WatcherConfig
	Service string // only if Compose is true
	Run     string
	Clean   string
	Verbose bool
}

// Basic Flags
type RootFlags struct {
	WC      WatcherConfig
	Build   []string
	Run     string
	Verbose bool
}
