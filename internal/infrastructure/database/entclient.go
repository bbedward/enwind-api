package database

import (
	"database/sql"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/bbedward/enwind-api/ent"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

func NewEntClient(connInfo SqlDBConn) (*ent.Client, *sql.DB, error) {
	db, err := sql.Open(connInfo.Driver(), connInfo.DSN())
	if err != nil {
		return nil, nil, err
	}

	drv := entsql.OpenDB(connInfo.Dialect(), db)
	return ent.NewClient(ent.Driver(drv)), db, nil
}
