package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Job represents the jobs table
// DBConfig groups the connection parameters that are _not_ persisted
// with a job. The frontend must supply these when starting a run so the
// worker can open source/target connections without storing credentials.
//
// Note the field names correspond to the JSON we expect from the client.
// We keep the `Type` field so the driver can be chosen at runtime.
//
// This struct lives in models so it can be shared by handlers, services,
// and workers without introducing import cycles.

type DBConfig struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// Job represents the jobs table
// connection parameters are intentionally omitted; they are supplied at run time
// instead of being stored in the database.
type Job struct {
	ID              string        `gorm:"type:char(36);primaryKey" json:"id"`
	Name            string        `json:"name"`
	SourceDBType    string        `json:"source_db_type"`
	TargetDBType    *string       `json:"target_db_type,omitempty"`
	TargetTableName *string       `json:"target_table_name,omitempty"`
	OutputType      string        `json:"output_type"`
	CreatedAt       time.Time     `json:"created_at"`
	JobTables       []JobTable    `gorm:"foreignKey:JobID" json:"tables,omitempty"`
	MaskingRules    []MaskingRule `gorm:"foreignKey:JobID" json:"rules,omitempty"`
	JobRuns         []JobRun      `gorm:"foreignKey:JobID" json:"runs,omitempty"`
}

// JobTable represents job_tables
type JobTable struct {
	ID        string    `gorm:"type:char(36);primaryKey" json:"id"`
	JobID     string    `gorm:"type:char(36);index" json:"job_id"`
	TableName string    `json:"table_name"`
	CreatedAt time.Time `json:"created_at"`
}

// MaskingRule represents masking_rules
type MaskingRule struct {
	ID         string         `gorm:"type:char(36);primaryKey" json:"id"`
	JobID      string         `gorm:"type:char(36);index" json:"job_id"`
	ColumnName string         `json:"column_name"`
	MaskType   string         `json:"mask_type"`
	Parameters datatypes.JSON `json:"parameters"`
	CreatedAt  time.Time      `json:"created_at"`
}

// JobRun represents job_runs
type JobRun struct {
	ID            string     `gorm:"type:char(36);primaryKey" json:"id"`
	JobID         string     `gorm:"type:char(36);index" json:"job_id"`
	Status        string     `json:"status"`
	RowsProcessed int        `json:"rows_processed"`
	StartedAt     time.Time  `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at,omitempty"`
	Log           string     `json:"log"`
}

// before create hooks assign UUIDs if missing
func (j *Job) BeforeCreate(tx *gorm.DB) (err error) {
	if j.ID == "" {
		j.ID = uuid.New().String()
	}
	return
}

func (jt *JobTable) BeforeCreate(tx *gorm.DB) (err error) {
	if jt.ID == "" {
		jt.ID = uuid.New().String()
	}
	return
}

func (r *MaskingRule) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return
}

func (r *JobRun) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	return
}
