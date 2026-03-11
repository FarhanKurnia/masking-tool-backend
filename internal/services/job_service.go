package services

import (
	"errors"
	"time"

	"github.com/example/masking-tool-backend/internal/models"
	"github.com/example/masking-tool-backend/internal/repositories"
	"github.com/example/masking-tool-backend/internal/workers"
	"github.com/sirupsen/logrus"
)

const DefaultPageSize = 20

// JobService defines business logic
type JobService interface {
	// CreateJob persists a job configuration without any sensitive
	// connection details; those are supplied later when the job is run.
	CreateJob(name string, sourceType string, table string, output string, targetType *string, targetTableName *string) (*models.Job, error)
	ListJobs(page, pageSize int) ([]models.Job, int64, error)
	SaveRules(jobID string, rules []models.MaskingRule) error
	// RunJob now requires connection configs at invocation time.
	RunJob(jobID string, src models.DBConfig, tgt *models.DBConfig) (*models.JobRun, error)
	GetRunStatus(runID string) (*models.JobRun, error)
}

type jobService struct {
	repo repositories.JobRepository
}

func NewJobService(repo repositories.JobRepository) JobService {
	return &jobService{repo: repo}
}

func (s *jobService) CreateJob(name string, sourceType string, table string, output string, targetType *string, targetTableName *string) (*models.Job, error) {
	// accept legacy output value
	if output == "staging" {
		output = "target"
	}
	job := &models.Job{
		Name:            name,
		SourceDBType:    sourceType,
		TargetDBType:    targetType,
		TargetTableName: targetTableName,
		OutputType:      output,
		CreatedAt:       time.Now(),
	}

	if err := s.repo.Create(job); err != nil {
		return nil, err
	}
	// create job table entry
	jt := models.JobTable{JobID: job.ID, TableName: table, CreatedAt: time.Now()}
	if err := s.repo.SaveJobTable(&jt); err != nil {
		logrus.Warn("failed to save job_table", err)
	}
	return job, nil
}

func (s *jobService) ListJobs(page, pageSize int) ([]models.Job, int64, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize
	jobs, count, err := s.repo.List(offset, pageSize)
	return jobs, count, err
}

func (s *jobService) SaveRules(jobID string, rules []models.MaskingRule) error {
	return s.repo.SaveMaskingRules(jobID, rules)
}

func (s *jobService) RunJob(jobID string, src models.DBConfig, tgt *models.DBConfig) (*models.JobRun, error) {
	job, err := s.repo.GetByID(jobID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, errors.New("job not found")
	}

	run := &models.JobRun{
		JobID:      jobID,
		Status:     "running",
		StartedAt:  time.Now(),
		FinishedAt: nil,
	}
	if err := s.repo.CreateJobRun(run); err != nil {
		return nil, err
	}
	// launch worker with supplied configs
	go func(runID string, sCfg models.DBConfig, tCfg *models.DBConfig) {
		if err := workers.ExecuteJob(runID, sCfg, tCfg); err != nil {
			logrus.Error(err)
		}
	}(run.ID, src, tgt)
	return run, nil
}

func (s *jobService) GetRunStatus(runID string) (*models.JobRun, error) {
	return s.repo.GetJobRun(runID)
}
