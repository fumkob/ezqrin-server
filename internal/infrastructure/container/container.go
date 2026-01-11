package container

import (
	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/domain/repository"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache"
	redisClient "github.com/fumkob/ezqrin-server/internal/infrastructure/cache/redis"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/internal/usecase/auth"
	"github.com/fumkob/ezqrin-server/internal/usecase/event"
	"github.com/fumkob/ezqrin-server/pkg/logger"
)

// Container holds all application dependencies
type Container struct {
	Repositories *RepositoryContainer
	UseCases     *UseCaseContainer
}

// RepositoryContainer holds repository implementations
type RepositoryContainer struct {
	User      repository.UserRepository
	Event     repository.EventRepository
	Blacklist repository.TokenBlacklistRepository
}

// UseCaseContainer holds use case orchestrators
type UseCaseContainer struct {
	Auth  *AuthUseCases
	Event event.Usecase
}

// AuthUseCases holds authentication-related use cases
type AuthUseCases struct {
	Register *auth.RegisterUseCase
	Login    *auth.LoginUseCase
	Refresh  *auth.RefreshTokenUseCase
	Logout   *auth.LogoutUseCase
}

// NewContainer initializes and wires all application dependencies
func NewContainer(
	cfg *config.Config,
	logger *logger.Logger,
	db database.Service,
	cache cache.Service,
) *Container {
	// Initialize repositories
	repos := &RepositoryContainer{
		User:  database.NewUserRepository(db.GetPool(), logger),
		Event: database.NewEventRepository(db.GetPool(), logger),
	}

	// TokenBlacklistRepository comes from Redis client
	if redis, ok := cache.(*redisClient.Client); ok {
		repos.Blacklist = redisClient.NewTokenBlacklistRepository(redis)
	}

	// Initialize use cases
	useCases := &UseCaseContainer{
		Auth: &AuthUseCases{
			Register: auth.NewRegisterUseCase(repos.User, cfg.JWT.Secret, logger),
			Login:    auth.NewLoginUseCase(repos.User, cfg.JWT.Secret, logger),
			Refresh:  auth.NewRefreshTokenUseCase(repos.User, repos.Blacklist, cfg.JWT.Secret, logger),
			Logout:   auth.NewLogoutUseCase(repos.Blacklist, cfg.JWT.Secret, logger),
		},
		Event: event.NewUsecase(repos.Event),
	}

	return &Container{
		Repositories: repos,
		UseCases:     useCases,
	}
}
