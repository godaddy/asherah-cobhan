package asherah

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

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
