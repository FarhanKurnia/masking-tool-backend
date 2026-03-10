package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Job represents the jobs table
type Job struct {
	ID                string        `gorm:"type:char(36);primaryKey" json:"id"`
	Name              string        `json:"name"`
	SourceDBType      string        `json:"source_db_type"`
	SourceDBHost      string        `json:"source_db_host"`
	SourceDBPort      int           `json:"source_db_port"`
	SourceDBName      string        `json:"source_db_name"`
	SourceDBUser      string        `json:"source_db_user"`
	SourceDBPassword  string        `json:"-"`
	StagingDBType     *string       `json:"staging_db_type,omitempty"`
	StagingDBHost     *string       `json:"staging_db_host,omitempty"`
	StagingDBPort     *int          `json:"staging_db_port,omitempty"`
	StagingDBName     *string       `json:"staging_db_name,omitempty"`
	StagingDBUser     *string       `json:"staging_db_user,omitempty"`
	StagingDBPassword *string       `json:"-"`
	StagingTableName  *string       `json:"staging_table_name,omitempty"`
	OutputType        string        `json:"output_type"`
	CreatedAt         time.Time     `json:"created_at"`
	JobTables         []JobTable    `gorm:"foreignKey:JobID" json:"tables,omitempty"`
	MaskingRules      []MaskingRule `gorm:"foreignKey:JobID" json:"rules,omitempty"`
	JobRuns           []JobRun      `gorm:"foreignKey:JobID" json:"runs,omitempty"`
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
	ID            string    `gorm:"type:char(36);primaryKey" json:"id"`
	JobID         string    `gorm:"type:char(36);index" json:"job_id"`
	Status        string    `json:"status"`
	RowsProcessed int       `json:"rows_processed"`
	StartedAt     time.Time `json:"started_at"`
	// FinishedAt    *time.Time `db:"finished_at" json:"finished_at"`
	FinishedAt time.Time `json:"finished_at"`
	Log        string    `json:"log"`
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
