package database

import (
	"fmt"
	"os"

	"entgo.io/ent/dialect"
	"github.com/bbedward/enwind-api/config"
	"github.com/bbedward/enwind-api/internal/common/log"
)

type SqlDBConn interface {
	DSN() string
	Dialect() string
	Driver() string
}

type PostgresConn struct {
	Host     string
	Port     int
	Password string
	User     string
	DBName   string
	SSLMode  string
}

func (c *PostgresConn) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", c.User, c.Password, c.Host, c.Port, c.DBName, c.SSLMode)
}

func (c *PostgresConn) Dialect() string {
	return dialect.Postgres
}

func (c *PostgresConn) Driver() string {
	return "pgx"
}

type SqliteConn struct {
	FileName string
	Mode     string
}

func (c *SqliteConn) DSN() string {
	return fmt.Sprintf("file:%s?cache=shared&mode=%s&_fk=1", c.FileName, c.Mode)
}

func (c *SqliteConn) Dialect() string {
	return dialect.SQLite
}

func (c *SqliteConn) Driver() string { return "sqlite" }

// Gets the DB connection information based on environment variables
func GetSqlDbConn(cfg *config.Config, mock bool) (SqlDBConn, error) {
	if mock {
		return &SqliteConn{FileName: "testing", Mode: "memory"}, nil
	}
	// Use postgres
	postgresDb := cfg.PostgresDB
	postgresUser := cfg.PostgresUser
	postgresPassword := cfg.PostgresPassword
	postgresHost := cfg.PostgresHost
	postgresPort := cfg.PostgresPort

	if postgresDb == "" || postgresUser == "" || postgresPassword == "" {
		log.Error("Postgres environment variables not set, not sure what to do? so exiting")
		os.Exit(1)
	}

	sslMode := cfg.PostgresSSLMode
	if sslMode == "" {
		sslMode = "disable" // Default to disable if not set
	}

	return &PostgresConn{
		Host:     postgresHost,
		Port:     postgresPort,
		Password: postgresPassword,
		User:     postgresUser,
		DBName:   postgresDb,
		SSLMode:  sslMode,
	}, nil
}
