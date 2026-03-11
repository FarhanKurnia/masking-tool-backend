package workers

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/example/masking-tool-backend/internal/masking"
	"github.com/example/masking-tool-backend/internal/models"
	"github.com/example/masking-tool-backend/internal/repositories"
	"github.com/example/masking-tool-backend/pkg/connection"
	"github.com/example/masking-tool-backend/pkg/db"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// ColumnInfo holds complete schema information about a column
type ColumnInfo struct {
	Name               string
	DataType           string // e.g., "varchar", "int", "decimal"
	ColumnType         string // Full type with constraints: "varchar(255)", "int(11) unsigned"
	IsNullable         bool
	ColumnKey          string  // "PRI", "UNI", "MUL", or ""
	Extra              string  // "auto_increment", "on update CURRENT_TIMESTAMP"
	CharacterMaxLength *int    // For VARCHAR, CHAR, TEXT
	NumericPrecision   *int    // For DECIMAL, NUMERIC, INT, etc.
	NumericScale       *int    // For DECIMAL, NUMERIC
	ColumnDefault      *string // Default value
}

// processJob reads rows from source DB, applies masking rules, writes output based on job.OutputType
func processJob(job *models.Job, run *models.JobRun, srcCfg models.DBConfig, tgtCfg *models.DBConfig) error {
	// prepare mask map
	maskMap := make(map[string]models.MaskingRule)
	for _, r := range job.MaskingRules {
		maskMap[r.ColumnName] = r
	}

	if len(job.JobTables) == 0 {
		return fmt.Errorf("no table configured")
	}
	table := job.JobTables[0].TableName

	// connect to source using configuration supplied at run time
	srcDb, err := connection.Connect(srcCfg.Type, srcCfg.Host, srcCfg.Port, srcCfg.Database, srcCfg.User, srcCfg.Password)
	if err != nil {
		return err
	}
	defer srcDb.Close()

	ctx := context.Background()
	rows, err := srcDb.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	cols, _ := rows.Columns()

	if job.OutputType == "csv" {
		return processToCSV(job, run, rows, cols, maskMap, table)
	} else if job.OutputType == "target" || job.OutputType == "staging" {
		// accept both new and legacy values
		return processToTarget(job, run, rows, cols, maskMap, table, tgtCfg, srcCfg)
	} else {
		return fmt.Errorf("unsupported output type: %s", job.OutputType)
	}
}

func processToCSV(job *models.Job, run *models.JobRun, rows *sql.Rows, cols []string, maskMap map[string]models.MaskingRule, table string) error {
	file, err := os.Create("output.csv")
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	writer.Write(cols)

	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}

	batch := 0

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}
		// apply masks
		for i, col := range cols {
			if rule, ok := maskMap[col]; ok {
				// convert value to string (handle []byte specially)
				var orig string
				if vals[i] != nil {
					switch v := vals[i].(type) {
					case []byte:
						orig = string(v)
					default:
						orig = fmt.Sprintf("%v", v)
					}
				}
				// filter rule.Parameters
				params := make(map[string]interface{})
				if len(rule.Parameters) > 0 {
					json.Unmarshal(rule.Parameters, &params)
				}
				masked := masking.MaskValue(orig, rule.MaskType, params)
				vals[i] = masked
			}
		}
		// write masked row
		strRow := make([]string, len(cols))
		for i, v := range vals {
			if v == nil {
				strRow[i] = ""
			} else {
				switch vv := v.(type) {
				case []byte:
					strRow[i] = string(vv)
				default:
					strRow[i] = fmt.Sprintf("%v", vv)
				}
			}
		}
		writer.Write(strRow)

		batch++
		run.RowsProcessed++
		if batch >= 1000 {
			run.Status = "running"
			repositories.NewJobRepository(db.Conn).UpdateJobRun(run)
			batch = 0
		}
	}

	// final update
	run.Status = "completed"
	now := time.Now()
	run.FinishedAt = &now
	duration := now.Sub(run.StartedAt)
	run.Log = fmt.Sprintf("duration=%s", duration)
	return repositories.NewJobRepository(db.Conn).UpdateJobRun(run)
}

func processToTarget(job *models.Job, run *models.JobRun, rows *sql.Rows, cols []string, maskMap map[string]models.MaskingRule, table string, tgtCfg *models.DBConfig, srcCfg models.DBConfig) error {
	if tgtCfg == nil {
		return fmt.Errorf("target DB configuration missing")
	}

	dbType := tgtCfg.Type
	if dbType == "" {
		dbType = "mysql"
	}

	targetDb, err := connection.Connect(dbType, tgtCfg.Host, tgtCfg.Port, tgtCfg.Database, tgtCfg.User, tgtCfg.Password)
	if err != nil {
		return err
	}
	defer targetDb.Close()

	targetTable := table
	if job.TargetTableName != nil && *job.TargetTableName != "" {
		targetTable = *job.TargetTableName
	}

	// Get source table schema by reconnecting to source DB (use source config)
	srcDb, err := connection.Connect(srcCfg.Type, srcCfg.Host, srcCfg.Port, srcCfg.Database, srcCfg.User, srcCfg.Password)
	if err != nil {
		return fmt.Errorf("failed to connect to source DB for schema: %v", err)
	}
	defer srcDb.Close()

	columnSchemas, err := getSourceTableSchema(srcDb, srcCfg.Type, srcCfg.Database, table)
	if err != nil {
		return fmt.Errorf("failed to get source table schema: %v", err)
	}

	// Drop existing table and recreate with proper schema
	dropTableSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", targetTable)
	if _, err := targetDb.Exec(dropTableSQL); err != nil {
		// If drop fails, try to truncate instead
		truncateSQL := fmt.Sprintf("TRUNCATE TABLE %s", targetTable)
		if _, err := targetDb.Exec(truncateSQL); err != nil {
			// If truncate also fails, log but continue - table might not exist yet
			fmt.Printf("Warning: could not drop or truncate table %s: %v\n", targetTable, err)
		}
	}

	// Create table with proper schema
	createTableSQL := generateCreateTableSQL(targetTable, columnSchemas)
	_, err = targetDb.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	// Prepare insert and update statements
	insertSQL := generateInsertSQL(targetTable, cols)
	updateSQL := generateUpdateSQL(targetTable, cols)

	var insertStmt, updateStmt *sql.Stmt
	insertStmt, err = targetDb.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %v", err)
	}
	defer insertStmt.Close()

	updateStmt, err = targetDb.Prepare(updateSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %v", err)
	}
	defer updateStmt.Close()

	vals := make([]interface{}, len(cols))
	ptrs := make([]interface{}, len(cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}

	batch := 0
	var rowsToProcess [][]interface{}

	// Collect all rows first
	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return err
		}

		// Make a copy of vals to avoid pointer issues
		rowVals := make([]interface{}, len(vals))
		for i, v := range vals {
			if v == nil {
				rowVals[i] = nil
			} else {
				// convert byte slices to string so staging columns get text, not raw bytes
				switch vv := v.(type) {
				case []byte:
					rowVals[i] = string(vv)
				default:
					rowVals[i] = vv
				}
			}
		}

		// apply masks only to columns with masking rules
		for i, col := range cols {
			if rule, ok := maskMap[col]; ok {
				// convert value to string (handle []byte specially)
				var orig string
				if rowVals[i] != nil {
					switch v := rowVals[i].(type) {
					case []byte:
						orig = string(v)
					default:
						orig = fmt.Sprintf("%v", v)
					}
				}
				// filter rule.Parameters
				params := make(map[string]interface{})
				if len(rule.Parameters) > 0 {
					json.Unmarshal(rule.Parameters, &params)
				}
				masked := masking.MaskValue(orig, rule.MaskType, params)
				rowVals[i] = masked
			}
		}

		rowsToProcess = append(rowsToProcess, rowVals)
	}

	// Close rows iteration and check for errors
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating source rows: %v", err)
	}

	// Build map of existing IDs in staging table for quick lookups
	existingIDs := make(map[string]bool)
	if len(rowsToProcess) > 0 {
		queryExistingSQL := fmt.Sprintf("SELECT %s FROM %s", cols[0], targetTable)
		existingRows, err := targetDb.Query(queryExistingSQL)
		if err == nil {
			defer existingRows.Close()
			for existingRows.Next() {
				var id interface{}
				if err := existingRows.Scan(&id); err == nil {
					existingIDs[fmt.Sprintf("%v", id)] = true
				}
			}
		}
	}

	// Now insert or update each collected row
	for _, rowVals := range rowsToProcess {
		idValue := rowVals[0]
		if idValue == nil {
			return fmt.Errorf("ID value is NULL at row %d, cannot insert/update", run.RowsProcessed+1)
		}

		idStr := fmt.Sprintf("%v", idValue)

		if existingIDs[idStr] {
			// Row exists, update it (all columns except first, then ID at end)
			updateVals := rowVals[1:]                // All columns except ID
			updateVals = append(updateVals, idValue) // Add ID at the end for WHERE clause
			_, err := updateStmt.Exec(updateVals...)
			if err != nil {
				return fmt.Errorf("failed to update row %d (ID: %v): %v", run.RowsProcessed+1, idValue, err)
			}
		} else {
			// Row doesn't exist, insert it
			_, err := insertStmt.Exec(rowVals...)
			if err != nil {
				return fmt.Errorf("failed to insert row %d (ID: %v): %v", run.RowsProcessed+1, idValue, err)
			}
		}

		batch++
		run.RowsProcessed++
		if batch >= 1000 {
			run.Status = "running"
			repositories.NewJobRepository(db.Conn).UpdateJobRun(run)
			batch = 0
		}
	}

	// final update
	run.Status = "completed"
	now := time.Now()
	run.FinishedAt = &now
	duration := now.Sub(run.StartedAt)
	run.Log = fmt.Sprintf("duration=%s", duration)
	return repositories.NewJobRepository(db.Conn).UpdateJobRun(run)
}

func generateCreateTableSQL(table string, columns []ColumnInfo) string {
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", table)
	for i, col := range columns {
		// Start with column name and type
		colDef := fmt.Sprintf("  %s %s", col.Name, col.ColumnType)

		// Handle NULL/NOT NULL
		if !col.IsNullable && col.Extra != "auto_increment" {
			colDef += " NOT NULL"
		}

		// Add DEFAULT if present (handle special MySQL keywords)
		if col.ColumnDefault != nil && *col.ColumnDefault != "" {
			defaultVal := *col.ColumnDefault
			// List of MySQL keywords that should not be quoted
			mysqlKeywords := map[string]bool{
				"CURRENT_TIMESTAMP":   true,
				"CURRENT_DATE":        true,
				"CURRENT_TIME":        true,
				"NULL":                true,
				"AUTO_INCREMENT":      true,
				"DEFAULT_GENERATED":   true,
				"(CURRENT_TIMESTAMP)": true,
				"(CURRENT_DATE)":      true,
				"(CURRENT_TIME)":      true,
				"CURRENT_TIMESTAMP()": true,
				"CURRENT_DATE()":      true,
				"CURRENT_TIME()":      true,
				"UUID()":              true,
				"RAND()":              true,
			}

			// Check if default is a keyword/function or needs quoting
			isKeyword := mysqlKeywords[defaultVal]
			// avoid copying over SQL expressions or functions that aren't valid as
			// column defaults in the target. these often show up as UUID_SHORT() in
			// our own tables and will trigger "Invalid default value" errors.
			if strings.Contains(defaultVal, "(") || strings.Contains(defaultVal, ")") || strings.Contains(strings.ToUpper(defaultVal), "UUID_SHORT") {
				defaultVal = ""
			}
			if defaultVal != "" {
				if !isKeyword && defaultVal != "NULL" {
					// Quote string defaults, but handle numeric defaults
					if _, err := strconv.Atoi(defaultVal); err != nil {
						// Not a number, so quote it
						defaultVal = fmt.Sprintf("'%s'", defaultVal)
					}
				}
				colDef += fmt.Sprintf(" DEFAULT %s", defaultVal)
			}
		}

		// Add EXTRA (AUTO_INCREMENT, etc.) - but skip if already in DEFAULT
		if col.Extra != "" && !strings.Contains(col.Extra, "DEFAULT") {
			colDef += fmt.Sprintf(" %s", col.Extra)
		}

		// Add PRIMARY KEY constraint
		if col.ColumnKey == "PRI" {
			colDef += " PRIMARY KEY"
		}

		sql += colDef
		if i < len(columns)-1 {
			sql += ",\n"
		}
	}
	sql += "\n)"
	return sql
}

func generateInsertSQL(table string, cols []string) string {
	sql := fmt.Sprintf("INSERT INTO %s (", table)
	for i, col := range cols {
		sql += col
		if i < len(cols)-1 {
			sql += ", "
		}
	}
	sql += ") VALUES ("
	for i := range cols {
		sql += "?"
		if i < len(cols)-1 {
			sql += ", "
		}
	}
	sql += ")"
	return sql
}

func generateUpdateSQL(table string, cols []string) string {
	if len(cols) == 0 {
		return ""
	}
	// UPDATE all columns except the first one (ID)
	sql := fmt.Sprintf("UPDATE %s SET ", table)
	for i := 1; i < len(cols); i++ {
		sql += fmt.Sprintf("%s = ?", cols[i])
		if i < len(cols)-1 {
			sql += ", "
		}
	}
	sql += fmt.Sprintf(" WHERE %s = ?", cols[0])
	return sql
}

// getSourceTableSchema fetches the complete schema from the source table
func getSourceTableSchema(srcDb *sql.DB, dbType, dbName, tableName string) ([]ColumnInfo, error) {
	var query string

	// Different query based on database type
	if dbType == "mysql" {
		query = `
			SELECT 
				COLUMN_NAME,
				DATA_TYPE,
				COLUMN_TYPE,
				IS_NULLABLE,
				COLUMN_KEY,
				EXTRA,
				CHARACTER_MAXIMUM_LENGTH,
				NUMERIC_PRECISION,
				NUMERIC_SCALE,
				COLUMN_DEFAULT
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
			ORDER BY ORDINAL_POSITION
		`
	} else if dbType == "postgres" {
		query = `
			SELECT 
				column_name,
				data_type,
				udt_name,
				is_nullable = 'YES',
				'',
				'',
				character_maximum_length,
				numeric_precision,
				numeric_scale,
				column_default
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = $1
			ORDER BY ordinal_position
		`
	} else {
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	var columns []ColumnInfo

	rows, err := srcDb.Query(query, dbName, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		var isNullableStr string
		var charMaxLen, numPrec, numScale sql.NullInt64
		var colDefault sql.NullString

		err := rows.Scan(
			&col.Name,
			&col.DataType,
			&col.ColumnType,
			&isNullableStr,
			&col.ColumnKey,
			&col.Extra,
			&charMaxLen,
			&numPrec,
			&numScale,
			&colDefault,
		)
		if err != nil {
			return nil, err
		}

		// Parse nullable
		col.IsNullable = (isNullableStr == "YES")

		// Handle nullable integers (convert int64 to int)
		if charMaxLen.Valid {
			val := int(charMaxLen.Int64)
			col.CharacterMaxLength = &val
		}
		if numPrec.Valid {
			val := int(numPrec.Int64)
			col.NumericPrecision = &val
		}
		if numScale.Valid {
			val := int(numScale.Int64)
			col.NumericScale = &val
		}
		if colDefault.Valid {
			col.ColumnDefault = &colDefault.String
		}

		columns = append(columns, col)
	}

	return columns, rows.Err()
}
