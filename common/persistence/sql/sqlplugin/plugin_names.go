package sqlplugin

// Plugin names live here so callers that only compare names do not link the
// drivers behind them.
const (
	MySQLPluginName         = "mysql8"
	PostgreSQLPluginName    = "postgres12"
	PostgreSQLPGXPluginName = "postgres12_pgx"
	SQLitePluginName        = "sqlite"
)
