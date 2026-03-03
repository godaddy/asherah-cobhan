package asherah

import (
	"database/sql"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var mysqlDSNPasswordRegexp = regexp.MustCompile(`^([^:]+):[^@]+@`)

const (
	ReplicaReadConsistencyQuery         = "SET aurora_replica_read_consistency = ?"
	ReplicaReadConsistencyValueEventual = "eventual"
	ReplicaReadConsistencyValueGlobal   = "global"
	ReplicaReadConsistencyValueSession  = "session"
)

var (
	dbconnection *sql.DB
)

func newConnection(dbdriver string, connStr string) (*sql.DB, error) {
	var err error
	if dbconnection == nil {
		dbconnection, err = sql.Open(dbdriver, connStr)
		if err != nil {
			return nil, err
		}
	}

	return dbconnection, nil
}

func redactConnectionString(connStr string) string {
	// URL-style: scheme://user:pass@host/db
	if idx := strings.Index(connStr, "://"); idx >= 0 {
		rest := connStr[idx+3:]
		if atIdx := strings.Index(rest, "@"); atIdx >= 0 {
			userInfo := rest[:atIdx]
			if colonIdx := strings.Index(userInfo, ":"); colonIdx >= 0 {
				return connStr[:idx+3] + userInfo[:colonIdx] + ":***@" + rest[atIdx+1:]
			}
		}
		return connStr
	}
	// MySQL DSN-style: user:pass@tcp(host)/db
	return mysqlDSNPasswordRegexp.ReplaceAllString(connStr, "${1}:***@")
}

func setRdbmsReplicaReadConsistencyValue(value string) (err error) {
	if dbconnection != nil {
		switch value {
		case
			ReplicaReadConsistencyValueEventual,
			ReplicaReadConsistencyValueGlobal,
			ReplicaReadConsistencyValueSession:
			_, err = dbconnection.Exec(ReplicaReadConsistencyQuery, value)
		}
	}

	return
}
