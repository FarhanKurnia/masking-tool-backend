package repositories

import (
	"github.com/example/masking-tool-backend/internal/models"
	"gorm.io/gorm"
)

// JobRepository defines methods for job data access
type JobRepository interface {
	Create(job *models.Job) error
	List(offset, limit int) ([]models.Job, int64, error)
	GetByID(id string) (*models.Job, error)
	SaveMaskingRules(jobID string, rules []models.MaskingRule) error
	CreateJobRun(run *models.JobRun) error
	UpdateJobRun(run *models.JobRun) error
	GetJobRun(id string) (*models.JobRun, error)
	SaveJobTable(table *models.JobTable) error
}

// jobRepo is gorm implementation
type jobRepo struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) JobRepository {
	return &jobRepo{db: db}
}

func (r *jobRepo) Create(job *models.Job) error {
	return r.db.Create(job).Error
}

func (r *jobRepo) List(offset, limit int) ([]models.Job, int64, error) {
	var jobs []models.Job
	var count int64
	if err := r.db.Model(&models.Job{}).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	// order by newest jobs first
	if err := r.db.Preload("JobTables").Preload("MaskingRules").Preload("JobRuns").Order("created_at DESC").Offset(offset).Limit(limit).Find(&jobs).Error; err != nil {
		return nil, 0, err
	}
	return jobs, count, nil
}

func (r *jobRepo) GetByID(id string) (*models.Job, error) {
	var job models.Job
	if err := r.db.Preload("JobTables").Preload("MaskingRules").Preload("JobRuns").First(&job, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *jobRepo) SaveMaskingRules(jobID string, rules []models.MaskingRule) error {
	if err := r.db.Where("job_id = ?", jobID).Delete(&models.MaskingRule{}).Error; err != nil {
		return err
	}
	for i := range rules {
		rules[i].JobID = jobID
	}
	return r.db.Create(&rules).Error
}

func (r *jobRepo) CreateJobRun(run *models.JobRun) error {
	return r.db.Create(run).Error
}

func (r *jobRepo) UpdateJobRun(run *models.JobRun) error {
	return r.db.Save(run).Error
}

func (r *jobRepo) GetJobRun(id string) (*models.JobRun, error) {
	var run models.JobRun
	if err := r.db.First(&run, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *jobRepo) SaveJobTable(table *models.JobTable) error {
	return r.db.Create(table).Error
}
