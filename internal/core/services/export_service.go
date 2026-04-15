package services

import (
	"context"

	"duskforge-api/internal/core/domain"
	"duskforge-api/internal/core/ports"

	"github.com/google/uuid"
)

type exportService struct {
	exportRepo ports.ExportRepository
}

func NewExportService(exportRepo ports.ExportRepository) ports.ExportService {
	return &exportService{exportRepo: exportRepo}
}

func (s *exportService) ExportUserData(ctx context.Context, userID uuid.UUID) (*domain.UserDataExport, error) {
	return s.exportRepo.GetAllUserData(ctx, userID)
}
