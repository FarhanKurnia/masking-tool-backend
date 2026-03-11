package handlers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/example/masking-tool-backend/internal/models"
	"github.com/example/masking-tool-backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"gorm.io/datatypes"
)

// JobHandler holds service reference
type JobHandler struct {
	svc services.JobService
}

func NewJobHandler(svc services.JobService) *JobHandler {
	return &JobHandler{svc: svc}
}

// CreateJob handler
func (h *JobHandler) CreateJob(c *fiber.Ctx) error {
	var req struct {
		Name     string `json:"name"`
		SourceDB struct {
			Type     string `json:"type"`
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Database string `json:"database"`
			User     string `json:"user"`
			Password string `json:"password"`
		} `json:"sourceDB"`
		Table    string `json:"table"`
		Output   string `json:"output"`
		TargetDB *struct {
			Type     string `json:"type"`
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Database string `json:"database"`
			User     string `json:"user"`
			Password string `json:"password"`
			Table    string `json:"table"`
		} `json:"targetDB,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}
	// we only need the target type and optional table name for storage;
	// other connection parameters will be provided later when running the job.
	var targetType, targetTableName *string
	if req.TargetDB != nil {
		typeVal := req.TargetDB.Type
		if typeVal == "" {
			typeVal = "mysql"
		}
		targetType = &typeVal
		if req.TargetDB.Table != "" {
			targetTableName = &req.TargetDB.Table
		}
	}
	job, err := h.svc.CreateJob(req.Name, req.SourceDB.Type, req.Table, req.Output, targetType, targetTableName)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": job.ID, "status": "created"})
}

// ListJobs handler with pagination
func (h *JobHandler) ListJobs(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	pageSize := services.DefaultPageSize
	jobs, total, err := h.svc.ListJobs(page, pageSize)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// transform to frontend-friendly shape
	type jobResp struct {
		ID         string     `json:"id"`
		TableName  string     `json:"table_name"`
		Status     string     `json:"status"`
		TotalRows  int        `json:"total_rows"`
		StartedAt  *time.Time `json:"started_at,omitempty"`
		FinishedAt *time.Time `json:"finished_at,omitempty"`
	}
	var out []jobResp
	for _, j := range jobs {
		table := ""
		if len(j.JobTables) > 0 {
			table = j.JobTables[0].TableName
		}
		status := "PENDING"
		rows := 0
		var startedAt *time.Time
		var finishedAt *time.Time
		if len(j.JobRuns) > 0 {
			latest := j.JobRuns[0]
			for _, r := range j.JobRuns {
				if r.StartedAt.After(latest.StartedAt) {
					latest = r
				}
			}
			status = strings.ToUpper(latest.Status)
			rows = latest.RowsProcessed
			startedAt = &latest.StartedAt
			finishedAt = latest.FinishedAt
		}
		out = append(out, jobResp{ID: j.ID, TableName: table, Status: status, TotalRows: rows, StartedAt: startedAt, FinishedAt: finishedAt})
	}

	return c.JSON(fiber.Map{"data": out, "total": total})
}

// SaveMaskingRules
func (h *JobHandler) SaveMaskingRules(c *fiber.Ctx) error {
	id := c.Params("id")
	var req struct {
		Rules []struct {
			Column     string                 `json:"column"`
			Type       string                 `json:"type"`
			Parameters map[string]interface{} `json:"parameters"`
		} `json:"rules"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}
	var rules []models.MaskingRule
	for _, r := range req.Rules {
		jsonParams, _ := json.Marshal(r.Parameters)
		rules = append(rules, models.MaskingRule{
			ColumnName: r.Column,
			MaskType:   r.Type,
			Parameters: datatypes.JSON(jsonParams),
		})
	}
	if err := h.svc.SaveRules(id, rules); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"status": "ok"})
}

// RunJob
func (h *JobHandler) RunJob(c *fiber.Ctx) error {
	id := c.Params("id")

	// parse connection information from request body
	var req struct {
		SourceDB struct {
			Type     string `json:"type"`
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Database string `json:"database"`
			User     string `json:"user"`
			Password string `json:"password"`
		} `json:"sourceDB"`
		TargetDB *struct {
			Type     string `json:"type"`
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Database string `json:"database"`
			User     string `json:"user"`
			Password string `json:"password"`
			Table    string `json:"table"`
		} `json:"targetDB,omitempty"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}

	srcCfg := models.DBConfig{
		Type:     req.SourceDB.Type,
		Host:     req.SourceDB.Host,
		Port:     req.SourceDB.Port,
		Database: req.SourceDB.Database,
		User:     req.SourceDB.User,
		Password: req.SourceDB.Password,
	}
	var tgtCfg *models.DBConfig
	if req.TargetDB != nil {
		tgtCfg = &models.DBConfig{
			Type:     req.TargetDB.Type,
			Host:     req.TargetDB.Host,
			Port:     req.TargetDB.Port,
			Database: req.TargetDB.Database,
			User:     req.TargetDB.User,
			Password: req.TargetDB.Password,
		}
		if req.TargetDB.Table != "" {
			tgtCfg.Database = req.TargetDB.Database
		}
	}

	run, err := h.svc.RunJob(id, srcCfg, tgtCfg)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"run_id": run.ID, "status": run.Status})
}

// DownloadCSV downloads the output CSV for a completed job
func (h *JobHandler) DownloadCSV(c *fiber.Ctx) error {
	runID := c.Params("runId")
	run, err := h.svc.GetRunStatus(runID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if run == nil || run.Status != "completed" {
		return fiber.NewError(fiber.StatusNotFound, "run not found or not completed")
	}
	return c.SendFile("output.csv")
}

// GetMaskingTypes returns the available masking strategies
func (h *JobHandler) GetMaskingTypes(c *fiber.Ctx) error {
	types := []map[string]string{
		{"id": "full", "name": "Full"},
		{"id": "partial", "name": "Partial"},
		{"id": "email", "name": "Email"},
		{"id": "name", "name": "Name"},
		{"id": "random_string", "name": "Random String"},
		{"id": "random_number", "name": "Random Number"},
		{"id": "null", "name": "Null"},
		{"id": "hash", "name": "Hash"},
	}
	return c.JSON(fiber.Map{"data": types})
}

// GetRunStatus
func (h *JobHandler) GetRunStatus(c *fiber.Ctx) error {
	id := c.Params("run_id")
	run, err := h.svc.GetRunStatus(id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(run)
}
