package workers

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/example/masking-tool-backend/internal/masking"
	"github.com/example/masking-tool-backend/internal/models"
	"github.com/example/masking-tool-backend/internal/repositories"
	"github.com/example/masking-tool-backend/pkg/connection"
	"github.com/example/masking-tool-backend/pkg/db"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// processJob reads rows from source DB, applies masking rules, writes output based on job.OutputType
func processJob(job *models.Job, run *models.JobRun) error {
	// prepare mask map
	maskMap := make(map[string]models.MaskingRule)
	for _, r := range job.MaskingRules {
		maskMap[r.ColumnName] = r
	}

	if len(job.JobTables) == 0 {
		return fmt.Errorf("no table configured")
	}
	table := job.JobTables[0].TableName

	srcDb, err := connection.Connect(job.SourceDBType, job.SourceDBHost, job.SourceDBPort, job.SourceDBName, job.SourceDBUser, job.SourceDBPassword)
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
	} else if job.OutputType == "staging" {
		return processToStaging(job, run, rows, cols, maskMap, table)
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
			run.FinishedAt = time.Now()
			repositories.NewJobRepository(db.Conn).UpdateJobRun(run)
			batch = 0
		}
	}

	// final update
	run.Status = "completed"
	run.FinishedAt = time.Now()
	duration := run.FinishedAt.Sub(run.StartedAt)
	run.Log = fmt.Sprintf("duration=%s", duration)
	return repositories.NewJobRepository(db.Conn).UpdateJobRun(run)
}

func processToStaging(job *models.Job, run *models.JobRun, rows *sql.Rows, cols []string, maskMap map[string]models.MaskingRule, table string) error {
	if job.StagingDBType == nil || job.StagingDBHost == nil || job.StagingDBPort == nil || job.StagingDBName == nil || job.StagingDBUser == nil || job.StagingDBPassword == nil {
		return fmt.Errorf("staging DB configuration missing")
	}

	dbType := *job.StagingDBType
	if dbType == "" {
		dbType = "mysql"
	}

	stagingDb, err := connection.Connect(dbType, *job.StagingDBHost, *job.StagingDBPort, *job.StagingDBName, *job.StagingDBUser, *job.StagingDBPassword)
	if err != nil {
		return err
	}
	defer stagingDb.Close()

	stagingTable := table
	if job.StagingTableName != nil && *job.StagingTableName != "" {
		stagingTable = *job.StagingTableName
	}

	// Drop existing table and recreate with proper schema
	dropTableSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", stagingTable)
	if _, err := stagingDb.Exec(dropTableSQL); err != nil {
		// If drop fails, try to truncate instead
		truncateSQL := fmt.Sprintf("TRUNCATE TABLE %s", stagingTable)
		if _, err := stagingDb.Exec(truncateSQL); err != nil {
			// If truncate also fails, log but continue - table might not exist yet
			fmt.Printf("Warning: could not drop or truncate table %s: %v\n", stagingTable, err)
		}
	}

	// Create table with PRIMARY KEY
	createTableSQL := generateCreateTableSQL(stagingTable, cols)
	_, err = stagingDb.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	// Prepare insert and update statements
	insertSQL := generateInsertSQL(stagingTable, cols)
	updateSQL := generateUpdateSQL(stagingTable, cols)

	var insertStmt, updateStmt *sql.Stmt
	insertStmt, err = stagingDb.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %v", err)
	}
	defer insertStmt.Close()

	updateStmt, err = stagingDb.Prepare(updateSQL)
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
		queryExistingSQL := fmt.Sprintf("SELECT %s FROM %s", cols[0], stagingTable)
		existingRows, err := stagingDb.Query(queryExistingSQL)
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
			run.FinishedAt = time.Now()
			repositories.NewJobRepository(db.Conn).UpdateJobRun(run)
			batch = 0
		}
	}

	// final update
	run.Status = "completed"
	run.FinishedAt = time.Now()
	duration := run.FinishedAt.Sub(run.StartedAt)
	run.Log = fmt.Sprintf("duration=%s", duration)
	return repositories.NewJobRepository(db.Conn).UpdateJobRun(run)
}

func generateCreateTableSQL(table string, cols []string) string {
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", table)
	for i, col := range cols {
		columnDef := fmt.Sprintf("%s TEXT", col)
		// Set first column as PRIMARY KEY (usually 'id')
		if i == 0 {
			columnDef = fmt.Sprintf("%s VARCHAR(255) PRIMARY KEY", col)
		}
		sql += columnDef
		if i < len(cols)-1 {
			sql += ", "
		}
	}
	sql += ")"
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
