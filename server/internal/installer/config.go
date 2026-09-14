package installer

type Config struct {
	Domain           string
	PostgresHost     string
	PostgresPort     string
	PostgresDatabase string
	PostgresUser     string
	PostgresPassword string

	InstallFreeSWITCH bool
	InstallRedis      bool
	InstallNATS       bool
}
