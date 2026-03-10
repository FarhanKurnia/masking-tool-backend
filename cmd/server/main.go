package main

import (
	"fmt"
	"log"

	"github.com/example/masking-tool-backend/config"
	"github.com/example/masking-tool-backend/internal/api"
	"github.com/example/masking-tool-backend/internal/models"
	"github.com/example/masking-tool-backend/pkg/db"
	"github.com/example/masking-tool-backend/pkg/logger"
)

func main() {
	logger.Init()

	if err := db.Init(); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// run auto migrations
	if err := db.Conn.AutoMigrate(
		&models.Job{}, &models.JobTable{}, &models.MaskingRule{}, &models.JobRun{},
	); err != nil {
		log.Fatalf("auto migrate failed: %v", err)
	}

	app := api.NewApp()
	port := config.Config.Port
	fmt.Printf("starting server on %s\n", port)
	log.Fatal(app.Listen("0.0.0.0:" + port))
}
