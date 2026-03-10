package workers

import (
	"time"

	"github.com/example/masking-tool-backend/internal/repositories"
	"github.com/example/masking-tool-backend/pkg/db"
)

// ExecuteJob performs masking job; uses processor implementation
func ExecuteJob(runID string) error {
	repo := repositories.NewJobRepository(db.Conn)
	run, err := repo.GetJobRun(runID)
	if err != nil {
		return err
	}
	if run == nil {
		return nil
	}

	job, err := repo.GetByID(run.JobID)
	if err != nil {
		return err
	}
	if job == nil {
		run.Status = "failed"
		run.Log = "job not found"
		run.FinishedAt = time.Now()
		return repo.UpdateJobRun(run)
	}

	// call processor that reads source and writes CSV
	if err := processJob(job, run); err != nil {
		run.Status = "failed"
		run.Log = err.Error()
	} else {
		run.Status = "completed"
		run.Log = "done"
	}
	run.FinishedAt = time.Now()
	return repo.UpdateJobRun(run)
}
