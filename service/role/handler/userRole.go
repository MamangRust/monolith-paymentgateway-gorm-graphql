package handler

import (
	"context"

	pbroles "github.com/MamangRust/monolith-payment-gateway-pb/role"
	pbuserroles "github.com/MamangRust/monolith-payment-gateway-pb/user_role"
	"github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-role/service"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	"github.com/MamangRust/monolith-payment-gateway-shared/errors"
	role_errors "github.com/MamangRust/monolith-payment-gateway-shared/errors/role_errors/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// userRoleHandleGrpc serves the UserRoleService. The user-role RPCs live in
// their own pb/user_role package but are registered on the same role gRPC
// server, so this handler shares the role service's query/command surfaces.
type userRoleHandleGrpc struct {
	pbuserroles.UnimplementedUserRoleServiceServer
	roleQuery   service.RoleQueryService
	roleCommand service.RoleCommandService
}

func NewUserRoleHandleGrpc(svc *service.Service) pbuserroles.UserRoleServiceServer {
	return &userRoleHandleGrpc{
		roleQuery:   svc.RoleQuery,
		roleCommand: svc.RoleCommand,
	}
}

func (s *userRoleHandleGrpc) FindByUserId(ctx context.Context, req *pbuserroles.FindByIdUserRoleRequest) (*pbroles.ApiResponsesRole, error) {
	userID := int(req.GetUserId())
	if userID <= 0 {
		return nil, role_errors.ErrGrpcRoleInvalidId
	}

	roles, err := s.roleQuery.FindByUserId(ctx, userID)
	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	return &pbroles.ApiResponsesRole{
		Status:  "success",
		Message: "Successfully fetched role by user id",
		Data:    mapResponsesRoleFromDB(roles),
	}, nil
}

func (s *userRoleHandleGrpc) AssignRoleToUser(ctx context.Context, req *pbuserroles.AssignRoleToUserRequest) (*pbuserroles.ApiResponseUserRole, error) {
	reqService := &requests.CreateUserRoleRequest{
		UserId: int(req.GetUserId()),
		RoleId: int(req.GetRoleId()),
	}

	userRole, err := s.roleCommand.AssignRoleToUser(ctx, reqService)
	if err != nil {
		return nil, errors.ToGrpcError(err)
	}

	return &pbuserroles.ApiResponseUserRole{
		Status:  "success",
		Message: "Successfully assigned role to user",
		Data:    mapResponseUserRole(userRole),
	}, nil
}

func (s *userRoleHandleGrpc) RemoveRoleFromUser(ctx context.Context, req *pbuserroles.RemoveRoleFromUserRequest) (*emptypb.Empty, error) {
	reqService := &requests.RemoveUserRoleRequest{
		UserId: int(req.GetUserId()),
		RoleId: int(req.GetRoleId()),
	}

	if err := s.roleCommand.RemoveRoleFromUser(ctx, reqService); err != nil {
		return nil, errors.ToGrpcError(err)
	}

	return &emptypb.Empty{}, nil
}

func mapResponsesRoleFromDB(roles []*models.Role) []*pbroles.RoleResponse {
	protoRoles := make([]*pbroles.RoleResponse, len(roles))
	for i, role := range roles {
		protoRoles[i] = &pbroles.RoleResponse{
			Id:        int32(role.RoleID),
			Name:      role.RoleName,
			CreatedAt: role.CreatedAt.Format("2006-01-02"),
			UpdatedAt: role.UpdatedAt.Format("2006-01-02"),
		}
	}
	return protoRoles
}

func mapResponseUserRole(userRole *models.UserRole) *pbuserroles.UserRoleResponse {
	if userRole == nil {
		return nil
	}
	return &pbuserroles.UserRoleResponse{
		UserRoleId: userRole.UserRoleID,
		UserId:     userRole.UserID,
		RoleId:     userRole.RoleID,
	}
}
