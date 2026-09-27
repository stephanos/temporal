package sql

import (
	"go.temporal.io/server/chasm"
	"go.temporal.io/server/common/namespace"
	"go.temporal.io/server/common/persistence/sql/sqlplugin"
	"go.temporal.io/server/common/searchattribute"
)

func NewQueryConverterLegacy(
	pluginName string,
	namespaceName namespace.Name,
	namespaceID namespace.ID,
	saTypeMap searchattribute.NameTypeMap,
	saMapper searchattribute.Mapper,
	queryString string,
	chasmMapper *chasm.VisibilitySearchAttributesMapper,
	archetypeID chasm.ArchetypeID,
) *QueryConverterLegacy {
	switch pluginName {
	case sqlplugin.MySQLPluginName:
		return newMySQLQueryConverter(namespaceName, namespaceID, saTypeMap, saMapper, queryString, chasmMapper, archetypeID)
	case sqlplugin.PostgreSQLPluginName, sqlplugin.PostgreSQLPGXPluginName:
		return newPostgreSQLQueryConverter(namespaceName, namespaceID, saTypeMap, saMapper, queryString, chasmMapper, archetypeID)
	case sqlplugin.SQLitePluginName:
		return newSqliteQueryConverter(namespaceName, namespaceID, saTypeMap, saMapper, queryString, chasmMapper, archetypeID)
	default:
		return nil
	}
}
