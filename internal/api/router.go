package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/example/masking-tool-backend/internal/handlers"
	"github.com/example/masking-tool-backend/internal/repositories"
	"github.com/example/masking-tool-backend/internal/services"
	"github.com/example/masking-tool-backend/pkg/db"
)

func NewApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			if e, ok := err.(*fiber.Error); ok {
				return c.Status(e.Code).JSON(fiber.Map{"error": e.Message})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(logger.New())

	// allow frontend dev server to access API
	app.Use(cors.New(cors.Config{
		AllowOrigins: "http://localhost:5173",
		AllowMethods: "GET,POST,HEAD,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	repo := repositories.NewJobRepository(db.Conn)
	svc := services.NewJobService(repo)
	handler := handlers.NewJobHandler(svc)
	dbh := handlers.NewDBHandler()

	api := app.Group("/api")

	// database introspection endpoints used by frontend wizard
	api.Post("/db/test-connection", dbh.TestConnection)
	api.Post("/db/tables", dbh.GetTables)
	api.Post("/db/columns", dbh.GetColumns)

	// masking metadata
	api.Get("/masking/types", handler.GetMaskingTypes)

	api.Post("/jobs", handler.CreateJob)
	api.Get("/jobs", handler.ListJobs)
	api.Post("/jobs/:id/masking-rules", handler.SaveMaskingRules)
	api.Post("/jobs/:id/run", handler.RunJob)
	api.Get("/runs/:run_id", handler.GetRunStatus)
	api.Get("/runs/:runId/download", handler.DownloadCSV)

	return app
}
