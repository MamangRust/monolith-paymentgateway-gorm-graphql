// Package user adapts the User service gRPC API into the shared domain model.
package user

import (
	"context"
	"time"

	pbuser "github.com/MamangRust/monolith-payment-gateway-pb/user"
	adapter "github.com/MamangRust/monolith-payment-gateway-pkg/adapter"
	models "github.com/MamangRust/monolith-payment-gateway-pkg/database/models"
	"github.com/MamangRust/monolith-payment-gateway-pkg/resilience"
	"github.com/MamangRust/monolith-payment-gateway-shared/domain/requests"
	user_errors "github.com/MamangRust/monolith-payment-gateway-shared/errors/user_errors/repository"
)

// QueryRepository is the contract consumers depend on for user reads.
type QueryRepository interface {
	FindById(ctx context.Context, user_id int) (*models.UserByIDRow, error)
	FindByEmail(ctx context.Context, email string) (*models.UserByEmailWithPasswordRow, error)
	FindByEmailAndVerify(ctx context.Context, email string) (*models.UserByEmailWithPasswordRow, error)
	FindByVerificationCode(ctx context.Context, code string) (*models.UserByVerificationCodeRow, error)
}

// CommandRepository is the contract consumers depend on for user writes.
type CommandRepository interface {
	CreateUser(ctx context.Context, req *requests.RegisterRequest) (*models.CreateUserRow, error)
	UpdateUserIsVerified(ctx context.Context, user_id int, is_verified bool) (*models.UserIsVerifiedRow, error)
	UpdateUserPassword(ctx context.Context, user_id int, password string) (*models.UserPasswordRow, error)
}

// Repository implements both QueryRepository and CommandRepository on top of the
// generated user query/command clients.
type Repository struct {
	query   pbuser.UserQueryServiceClient
	command pbuser.UserCommandServiceClient
	guard   *resilience.DependencyGuard
}

// SetGuard implements adapter.GuardSetter.
func (a *Repository) SetGuard(g *resilience.DependencyGuard) { a.guard = g }

// NewAdapter wraps the generated user clients. Resilience (timeout/circuit-breaker/
// bulkhead) is applied per call through the optional dependency guard; a nil
// guard is a passthrough.
func NewAdapter(query pbuser.UserQueryServiceClient, command pbuser.UserCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	a := &Repository{query: query, command: command}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// New wraps the generated user query/command clients. It is the canonical
// constructor; modules build it inside their repositories with the guard
// options they were given.
func New(query pbuser.UserQueryServiceClient, command pbuser.UserCommandServiceClient, opts ...adapter.GuardOption) *Repository {
	return NewAdapter(query, command, opts...)
}

// NewQueryAdapter returns the adapter restricted to the QueryRepository surface
// for consumers that only read.
func NewQueryAdapter(query pbuser.UserQueryServiceClient, opts ...adapter.GuardOption) QueryRepository {
	return NewAdapter(query, nil, opts...)
}

// NewCommandAdapter returns the adapter restricted to the CommandRepository
// surface for consumers that only write.
func NewCommandAdapter(command pbuser.UserCommandServiceClient, opts ...adapter.GuardOption) CommandRepository {
	return NewAdapter(nil, command, opts...)
}

func (a *Repository) FindById(ctx context.Context, user_id int) (*models.UserByIDRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbuser.ApiResponseUser, error) {
		return a.query.FindById(ctx, &pbuser.FindByIdUserRequest{Id: int32(user_id)})
	})
	if err != nil {
		return nil, user_errors.ErrUserNotFound.WithInternal(err)
	}
	if res == nil || res.Data == nil {
		return nil, user_errors.ErrUserNotFound
	}
	return &models.UserByIDRow{
		UserID:    res.Data.Id,
		Firstname: res.Data.Firstname,
		Lastname:  res.Data.Lastname,
		Email:     res.Data.Email,
		CreatedAt: parseTime(res.Data.CreatedAt),
		UpdatedAt: parseTime(res.Data.UpdatedAt),
	}, nil
}

func (a *Repository) FindByEmail(ctx context.Context, email string) (*models.UserByEmailWithPasswordRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbuser.ApiResponseUserWithPassword, error) {
		return a.query.FindByEmail(ctx, &pbuser.FindByEmailRequest{Email: email})
	})
	if err != nil {
		return nil, user_errors.ErrUserNotFound.WithInternal(err)
	}
	if res == nil || res.Data == nil {
		return nil, user_errors.ErrUserNotFound
	}
	return &models.UserByEmailWithPasswordRow{
		UserID:   res.Data.Id,
		Email:    res.Data.Email,
		Password: res.Data.Password,
	}, nil
}

// FindByEmailAndVerify looks up a verified user by email and surfaces the stored
// password so the auth login flow can compare hashes.
func (a *Repository) FindByEmailAndVerify(ctx context.Context, email string) (*models.UserByEmailWithPasswordRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbuser.ApiResponseUserWithPassword, error) {
		return a.query.FindByEmailAndVerify(ctx, &pbuser.FindByEmailAndVerifyRequest{Email: email})
	})
	if err != nil {
		return nil, user_errors.ErrUserNotFound.WithInternal(err)
	}
	if res == nil || res.Data == nil {
		return nil, user_errors.ErrUserNotFound
	}
	return &models.UserByEmailWithPasswordRow{
		UserID:   res.Data.Id,
		Email:    res.Data.Email,
		Password: res.Data.Password,
	}, nil
}

func (a *Repository) FindByVerificationCode(ctx context.Context, code string) (*models.UserByVerificationCodeRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbuser.ApiResponseUser, error) {
		return a.query.FindByVerificationCode(ctx, &pbuser.FindByVerificationCodeRequest{VerificationCode: code})
	})
	if err != nil {
		return nil, user_errors.ErrUserNotFound.WithInternal(err)
	}
	if res == nil || res.Data == nil {
		return nil, nil
	}
	return &models.UserByVerificationCodeRow{
		UserID:    res.Data.Id,
		Firstname: res.Data.Firstname,
		Lastname:  res.Data.Lastname,
		Email:     res.Data.Email,
		CreatedAt: parseTime(res.Data.CreatedAt),
		UpdatedAt: parseTime(res.Data.UpdatedAt),
	}, nil
}

func (a *Repository) CreateUser(ctx context.Context, req *requests.RegisterRequest) (*models.CreateUserRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbuser.ApiResponseUser, error) {
		return a.command.CreateUser(ctx, &pbuser.RegisterUserRequest{
			Firstname:        req.FirstName,
			Lastname:         req.LastName,
			Email:            req.Email,
			Password:         req.Password,
			VerificationCode: req.VerifiedCode,
			IsVerified:       req.IsVerified,
		})
	})
	if err != nil {
		return nil, user_errors.ErrCreateUser.WithInternal(err)
	}
	if res == nil || res.Data == nil {
		return nil, user_errors.ErrCreateUser
	}
	return &models.CreateUserRow{
		UserID:    res.Data.Id,
		Firstname: res.Data.Firstname,
		Lastname:  res.Data.Lastname,
		Email:     res.Data.Email,
		CreatedAt: parseTime(res.Data.CreatedAt),
		UpdatedAt: parseTime(res.Data.UpdatedAt),
	}, nil
}

func (a *Repository) UpdateUserIsVerified(ctx context.Context, user_id int, is_verified bool) (*models.UserIsVerifiedRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbuser.ApiResponseUser, error) {
		return a.command.UpdateUserIsVerified(ctx, &pbuser.UpdateUserIsVerifiedRequest{
			UserId:     int32(user_id),
			IsVerified: is_verified,
		})
	})
	if err != nil {
		return nil, user_errors.ErrUpdateUserVerificationCode.WithInternal(err)
	}
	if res == nil || res.Data == nil {
		return nil, user_errors.ErrUpdateUserVerificationCode
	}
	return &models.UserIsVerifiedRow{
		UserID:    res.Data.Id,
		Firstname: res.Data.Firstname,
		Lastname:  res.Data.Lastname,
		Email:     res.Data.Email,
		CreatedAt: parseTime(res.Data.CreatedAt),
		UpdatedAt: parseTime(res.Data.UpdatedAt),
	}, nil
}

func (a *Repository) UpdateUserPassword(ctx context.Context, user_id int, password string) (*models.UserPasswordRow, error) {
	res, err := adapter.Call(a.guard, ctx, func(ctx context.Context) (*pbuser.ApiResponseUser, error) {
		return a.command.UpdateUserPassword(ctx, &pbuser.UpdateUserPasswordRequest{
			UserId:   int32(user_id),
			Password: password,
		})
	})
	if err != nil {
		return nil, user_errors.ErrUpdateUserPassword.WithInternal(err)
	}
	if res == nil || res.Data == nil {
		return nil, user_errors.ErrUpdateUserPassword
	}
	return &models.UserPasswordRow{
		UserID:    res.Data.Id,
		Firstname: res.Data.Firstname,
		Lastname:  res.Data.Lastname,
		Email:     res.Data.Email,
		CreatedAt: parseTime(res.Data.CreatedAt),
		UpdatedAt: parseTime(res.Data.UpdatedAt),
	}, nil
}

// parseTime accepts the RFC3339 timestamps the user service emits and returns
// the zero time for empty or unparseable values.
func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
