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
	CreateJob(name string, sourceType string, host string, port int, database string, user string, password string, table string, output string, stagingType *string, stagingHost *string, stagingPort *int, stagingDatabase *string, stagingUser *string, stagingPassword *string, stagingTableName *string) (*models.Job, error)
	ListJobs(page, pageSize int) ([]models.Job, int64, error)
	SaveRules(jobID string, rules []models.MaskingRule) error
	RunJob(jobID string) (*models.JobRun, error)
	GetRunStatus(runID string) (*models.JobRun, error)
}

type jobService struct {
	repo repositories.JobRepository
}

func NewJobService(repo repositories.JobRepository) JobService {
	return &jobService{repo: repo}
}

func (s *jobService) CreateJob(name string, sourceType string, host string, port int, database string, user string, password string, table string, output string, stagingType *string, stagingHost *string, stagingPort *int, stagingDatabase *string, stagingUser *string, stagingPassword *string, stagingTableName *string) (*models.Job, error) {
	job := &models.Job{
		Name:              name,
		SourceDBType:      sourceType,
		SourceDBHost:      host,
		SourceDBPort:      port,
		SourceDBName:      database,
		SourceDBUser:      user,
		SourceDBPassword:  password,
		StagingDBType:     stagingType,
		StagingDBHost:     stagingHost,
		StagingDBPort:     stagingPort,
		StagingDBName:     stagingDatabase,
		StagingDBUser:     stagingUser,
		StagingDBPassword: stagingPassword,
		StagingTableName:  stagingTableName,
		OutputType:        output,
		CreatedAt:         time.Now(),
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

func (s *jobService) RunJob(jobID string) (*models.JobRun, error) {
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
		FinishedAt: time.Now(),
	}
	if err := s.repo.CreateJobRun(run); err != nil {
		return nil, err
	}
	go func(runID string) {
		if err := workers.ExecuteJob(runID); err != nil {
			logrus.Error(err)
		}
	}(run.ID)
	return run, nil
}

func (s *jobService) GetRunStatus(runID string) (*models.JobRun, error) {
	return s.repo.GetJobRun(runID)
}
