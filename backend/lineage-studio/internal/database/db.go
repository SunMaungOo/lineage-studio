package database

import (
	"database/sql"
	"fmt"
	"strings"

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

type ObjectType string

const (
	TypeView      ObjectType = "view"
	TypeProcedure ObjectType = "procedure"
)

type ObjectInfo struct {
	Schema     string
	Name       string
	Definition string
	ObjectType ObjectType
}

type Credential struct {
	User     string
	Password string
}

type HostInfo struct {
	Host     string
	Database string
}

func GetView(credential Credential, host HostInfo) ([]ObjectInfo, error) {
	return getObjectInfos(credential, host, viewSql, "view")
}

func GetProcedure(credential Credential, host HostInfo) ([]ObjectInfo, error) {
	return getObjectInfos(credential, host, procedureSql, "procedure")
}

func getObjectInfos(credential Credential, host HostInfo, objectInfoSql string, objectType ObjectType) ([]ObjectInfo, error) {

	var objects []ObjectInfo

	connectionString := fmt.Sprintf("sqlserver://%v:%v@%v?database=%v", credential.User, credential.Password, host.Host, host.Database)

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

		object.ObjectType = objectType

		objects = append(objects, object)

		count = count + 1

	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return objects, nil
}

func parseHostInfo(value string) (HostInfo, error) {

	if !strings.Contains(value, "/") {
		return HostInfo{}, fmt.Errorf("%v does not contain /", value)
	}

	lastIndex := strings.LastIndex(value, "/")

	return HostInfo{
		Host:     value[0:lastIndex],
		Database: value[lastIndex+1:],
	}, nil
}

func parseCredential(value string) (Credential, error) {

	if !strings.Contains(value, ":") {
		return Credential{}, fmt.Errorf("%v does not contain :", value)
	}

	lastIndex := strings.LastIndex(value, ":")

	return Credential{
		User:     value[0:lastIndex],
		Password: value[lastIndex+1:],
	}, nil

}

// format is user:password@host/database
func ParseDatabaseInfo(value string) (HostInfo, Credential, error) {

	if !strings.Contains(value, "@") {
		return HostInfo{}, Credential{}, fmt.Errorf("%v does not contain @", value)
	}

	index := strings.Index(value, "@")

	credentialStr := value[0:index]

	hostInfoStr := value[index+1:]

	hostInfo, err := parseHostInfo(hostInfoStr)

	if err != nil {
		return HostInfo{}, Credential{}, err
	}

	credential, err := parseCredential(credentialStr)

	if err != nil {
		return HostInfo{}, Credential{}, err
	}

	return hostInfo, credential, nil

}
