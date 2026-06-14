package database

import (
	"database/sql"
	"fmt"

	_ "github.com/microsoft/go-mssqldb"
)

const viewSql string = `

SELECT views.view_schema,
views.view_name,
sql_modules.definition AS view_definition
FROM 
(
	SELECT table_schema AS view_schema,
	table_name AS view_name,
	OBJECT_ID(CONCAT(table_schema,'.',table_name)) AS object_id
	FROM INFORMATION_SCHEMA.VIEWS
)AS views
INNER JOIN sys.sql_modules
ON views.object_id = sql_modules.object_id
ORDER BY views.view_schema,views.view_name;

`

const procedureSql string = `

SELECT routines.routine_schema,
routines.routine_name,
sql_modules.definition AS routines_definition
FROM
(
	SELECT routine_schema,
	routine_name,
	OBJECT_ID(CONCAT(routine_schema,'.',routine_name)) AS object_id
	FROM INFORMATION_SCHEMA.ROUTINES
	WHERE routine_type = 'PROCEDURE'
) AS routines
INNER JOIN sys.sql_modules
ON routines.object_id = sql_modules.object_id
ORDER BY routines.routine_schema,
routines.routine_name;

`

type ObjectInfo struct {
	Schema     string
	Name       string
	Definition string
}

func GetView(user string, password string, host string, database string) ([]ObjectInfo, error) {
	return getObjectInfos(user, password, host, database, viewSql)
}

func GetProcedure(user string, password string, host string, database string) ([]ObjectInfo, error) {
	return getObjectInfos(user, password, host, database, procedureSql)
}

func getObjectInfos(user string, password string, host string, database string, objectInfoSql string) ([]ObjectInfo, error) {

	var objects []ObjectInfo

	connectionString := fmt.Sprintf("sqlserver://%v:%v@%v?database=%v", user, password, host, database)

	db, err := sql.Open("sqlserver", connectionString)

	if err != nil {
		return nil, err
	}

	err = db.Ping()

	if err != nil {
		return nil, err
	}

	defer db.Close()

	rows, err := db.Query(objectInfoSql)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	count := 0

	for rows.Next() {

		var object ObjectInfo

		if err := rows.Scan(&object.Schema, &object.Name, &object.Definition); err != nil {
			return nil, err
		}

		objects = append(objects, object)

		count = count + 1

	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return objects, nil
}
