package handlers

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/example/masking-tool-backend/pkg/connection"
	"github.com/gofiber/fiber/v2"
)

// DBHandler provides endpoints for validating and introspecting source databases.
type DBHandler struct{}

func NewDBHandler() *DBHandler {
	return &DBHandler{}
}

// request payload used for all db endpoints
type dbReq struct {
	Type     string `json:"type" form:"type"` // e.g. "postgres", "mysql"
	Host     string `json:"host" form:"host"`
	Port     int    `json:"port" form:"port"`
	User     string `json:"user" form:"user"`
	Password string `json:"password" form:"password"`
	Database string `json:"database" form:"database"`
	Table    string `json:"table,omitempty" form:"table"`
}

// TestConnection opens a connection and pings the database.
func (h *DBHandler) TestConnection(c *fiber.Ctx) error {
	var req dbReq
	if err := c.BodyParser(&req); err != nil {
		log.Println("Error parsing DB connection request:", err)
		log.Println("Testing DB connection with config:", req)
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}
	db, err := connection.Connect(req.Type, req.Host, req.Port, req.Database, req.User, req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

// GetTables returns a list of table names in the specified schema / database.
func (h *DBHandler) GetTables(c *fiber.Ctx) error {
	var req dbReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}
	db, err := connection.Connect(req.Type, req.Host, req.Port, req.Database, req.User, req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer db.Close()

	var rows *sql.Rows
	if req.Type == "mysql" {
		rows, err = db.Query("SHOW TABLES")
	} else {
		rows, err = db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema='public'")
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	tables := []string{}
	for rows.Next() {
		var tbl string
		if err := rows.Scan(&tbl); err != nil {
			return err
		}
		tables = append(tables, tbl)
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"tables": tables}})
}

// GetColumns returns column names for a specific table.
func (h *DBHandler) GetColumns(c *fiber.Ctx) error {
	var req dbReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}
	db, err := connection.Connect(req.Type, req.Host, req.Port, req.Database, req.User, req.Password)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer db.Close()

	var rows *sql.Rows
	if req.Type == "mysql" {
		rows, err = db.Query(fmt.Sprintf("SHOW COLUMNS FROM %s", req.Table))
	} else {
		rows, err = db.Query(fmt.Sprintf("SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name='%s'", req.Table))
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer rows.Close()

	cols := []string{}
	for rows.Next() {
		var col string
		if req.Type == "mysql" {
			var dummy sql.RawBytes
			// SHOW COLUMNS returns multiple fields; first is Field name
			if err := rows.Scan(&col, &dummy, &dummy, &dummy, &dummy, &dummy); err != nil {
				return err
			}
		} else {
			if err := rows.Scan(&col); err != nil {
				return err
			}
		}
		cols = append(cols, col)
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"columns": cols}})
}
