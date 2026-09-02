package config

const (
	envDatabaseURL     = "DATABASE_URL"
	envPort            = "PORT"
	envServiceTimezone = "SERVICE_TIMEZONE"
	envMigrationsPath  = "MIGRATIONS_PATH"
)

const (
	defaultPort            = "8080"
	defaultServiceTimezone = "UTC"
	defaultMigrationsPath  = "file://migrations"
)
