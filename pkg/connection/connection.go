package connection

import (
	"database/sql"
	"fmt"
)

// NewSourceDB returns DSN for source database using provided config and type
func NewSourceDB(dbType, host string, port int, user, password, dbname string) string {
	if dbType == "mysql" {
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", user, password, host, port, dbname)
	}
	// postgres default
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", user, password, host, port, dbname)
}

// Connect opens a *sql.DB for the given database type and credentials
func Connect(dbType, host string, port int, dbname, user, password string) (*sql.DB, error) {
	var dsn string
	var driver string
	if dbType == "mysql" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true", user, password, host, port, dbname)
		driver = "mysql"
	} else {
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
		driver = "postgres"
	}
	return sql.Open(driver, dsn)
}
