package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/seoxpert/shared/models"
)

// ListProjects returns all projects owned by the authenticated user.
func (h *Handler) ListProjects(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := h.pool.Query(ctx, `
		SELECT id, user_id, name, domain, description, created_at, updated_at
		FROM projects
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 100
	`, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "db error"})
	}
	defer rows.Close()

	projects := make([]models.Project, 0, 10)
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Domain, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			continue
		}
		projects = append(projects, p)
	}

	return c.JSON(fiber.Map{"data": projects, "count": len(projects)})
}

// GetProject returns a single project by ID, scoped to the authenticated user.
func (h *Handler) GetProject(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	projectID := c.Params("id")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var p models.Project
	err := h.pool.QueryRow(ctx, `
		SELECT id, user_id, name, domain, description, created_at, updated_at
		FROM projects
		WHERE id = $1 AND user_id = $2
	`, projectID, userID).Scan(&p.ID, &p.UserID, &p.Name, &p.Domain, &p.Description, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	return c.JSON(fiber.Map{"data": p})
}

// CreateProject creates a new project after validating plan limits.
func (h *Handler) CreateProject(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)

	var req models.CreateProjectRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := validateProject(&req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Enforce plan project limit.
	plan, _ := h.rdb.Get(ctx, "plan:"+userID)
	if plan == "" {
		plan = "free"
	}
	limit := models.Limits[models.Plan(plan)].MaxProjects

	var count int
	if err := h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM projects WHERE user_id = $1`, userID).Scan(&count); err == nil {
		if count >= limit {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "project limit reached for your plan",
				"limit": limit,
				"plan":  plan,
			})
		}
	}

	var p models.Project
	err := h.pool.QueryRow(ctx, `
		INSERT INTO projects (user_id, name, domain, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, name, domain, description, created_at, updated_at
	`, userID, req.Name, req.Domain, req.Description).
		Scan(&p.ID, &p.UserID, &p.Name, &p.Domain, &p.Description, &p.CreatedAt, &p.UpdatedAt)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create project"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"data": p})
}

// DeleteProject removes a project and all associated data (cascade in SQL).
func (h *Handler) DeleteProject(c fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	projectID := c.Params("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tag, err := h.pool.Exec(ctx, `
		DELETE FROM projects WHERE id = $1 AND user_id = $2
	`, projectID, userID)

	if err != nil || tag.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "project not found"})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func validateProject(req *models.CreateProjectRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Domain = strings.TrimSpace(strings.ToLower(req.Domain))

	if len(req.Name) < 2 || len(req.Name) > 100 {
		return fiber.NewError(0, "name must be 2–100 characters")
	}
	if req.Domain == "" {
		return fiber.NewError(0, "domain is required")
	}
	if len(req.Description) > 500 {
		return fiber.NewError(0, "description must be under 500 characters")
	}
	return nil
}
