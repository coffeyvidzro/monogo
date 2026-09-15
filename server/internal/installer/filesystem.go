package installer

import "path/filepath"

const (
	InstallRoot = "/opt/leamout"
	ConfigRoot  = "/etc/leamout"
	StateRoot   = "/var/lib/leamout"
	LogRoot     = "/var/log/leamout"
	RunRoot     = "/run/leamout"

	EnvironmentPath = ConfigRoot + "/leamout.env"
	CertificateDir  = ConfigRoot + "/certs"
	LicenseDir      = ConfigRoot + "/license"
	BackupsDir      = StateRoot + "/backups"
	StagingDir      = StateRoot + "/staging"
	InstallStateDir = StateRoot + "/state"
	ACMEWebroot     = StateRoot + "/acme"
	CurrentPath     = InstallRoot + "/current"
	ReleasesDir     = InstallRoot + "/releases"
)

func ReleaseDir(version string) string {
	return filepath.Join(ReleasesDir, version)
}
