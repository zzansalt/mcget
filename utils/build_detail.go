package util

type BuildDetails struct {
	Name   string
	Size   int64
	SHA256 string
	URL    string
	// InstallArgs, when non-empty, are passed to `java -jar <Name>` after
	// downloading the installer to a temporary directory.
	InstallArgs []string
}
