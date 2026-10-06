package repository

import (
	"context"
	"errors"

	"github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	sharedErrors "github.com/MamangRust/monolith-payment-gateway-shared/errors"
	"gorm.io/gorm"
)

// AssignRoleToUser is idempotent: a repeated assignment returns the existing
// mapping instead of inserting a duplicate row.
func (r *roleCommandRepository) AssignRoleToUser(ctx context.Context, req *requests.CreateUserRoleRequest) (*models.UserRole, error) {
	userRole := &models.UserRole{
		UserID: int32(req.UserId),
		RoleID: int32(req.RoleId),
	}

	var existing models.UserRole
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND role_id = ?", userRole.UserID, userRole.RoleID).
		First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, sharedErrors.ErrNoRowsOrFailed(err, "UserRole", "assign role to user")
	}

	if err := r.db.WithContext(ctx).Create(userRole).Error; err != nil {
		return nil, sharedErrors.ErrConstraintOrFailed(err, "UserRole", "assign role to user")
	}
	return userRole, nil
}

func (r *roleCommandRepository) RemoveRoleFromUser(ctx context.Context, req *requests.RemoveUserRoleRequest) error {
	// The mapping is hard-deleted so a later re-assignment starts clean.
	if err := r.db.WithContext(ctx).Unscoped().
		Where("user_id = ? AND role_id = ?", req.UserId, req.RoleId).
		Delete(&models.UserRole{}).Error; err != nil {
		return sharedErrors.ErrNoRowsOrFailed(err, "UserRole", "remove role from user")
	}
	return nil
}
