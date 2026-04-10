package buildinfo

var (
	Version   = "dev"
	Commit    = "local"
	BuildDate = "unknown"
)

func DisplayVersion() string {
	if Version == "" {
		return "dev"
	}
	return Version
}
