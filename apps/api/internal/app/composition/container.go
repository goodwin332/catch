package composition

import (
	"context"
	"log/slog"
	"time"

	"catch/apps/api/internal/app/config"
	articlepg "catch/apps/api/internal/modules/articles/adapters/postgres"
	articleapp "catch/apps/api/internal/modules/articles/app"
	bookmarkpg "catch/apps/api/internal/modules/bookmarks/adapters/postgres"
	bookmarkapp "catch/apps/api/internal/modules/bookmarks/app"
	chatpg "catch/apps/api/internal/modules/chat/adapters/postgres"
	chatapp "catch/apps/api/internal/modules/chat/app"
	commentpg "catch/apps/api/internal/modules/comments/adapters/postgres"
	commentapp "catch/apps/api/internal/modules/comments/app"
	identitypg "catch/apps/api/internal/modules/identity/adapters/postgres"
	identityapp "catch/apps/api/internal/modules/identity/app"
	mediapg "catch/apps/api/internal/modules/media/adapters/postgres"
	mediastorage "catch/apps/api/internal/modules/media/adapters/storage"
	mediaapp "catch/apps/api/internal/modules/media/app"
	moderationpg "catch/apps/api/internal/modules/moderation/adapters/postgres"
	moderationapp "catch/apps/api/internal/modules/moderation/app"
	notificationpg "catch/apps/api/internal/modules/notifications/adapters/postgres"
	notificationapp "catch/apps/api/internal/modules/notifications/app"
	profilepg "catch/apps/api/internal/modules/profiles/adapters/postgres"
	profileapp "catch/apps/api/internal/modules/profiles/app"
	reactionpg "catch/apps/api/internal/modules/reactions/adapters/postgres"
	reactionapp "catch/apps/api/internal/modules/reactions/app"
	reportpg "catch/apps/api/internal/modules/reports/adapters/postgres"
	reportapp "catch/apps/api/internal/modules/reports/app"
	platformauth "catch/apps/api/internal/platform/auth"
	"catch/apps/api/internal/platform/db"
	platformoauth "catch/apps/api/internal/platform/oauth"
	"catch/apps/api/internal/platform/search"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Container struct {
	Config        config.Config
	Logger        *slog.Logger
	DB            *pgxpool.Pool
	TxManager     *db.TxManager
	Auth          *platformauth.Manager
	Identity      *identityapp.Service
	Profiles      *profileapp.Service
	Articles      *articleapp.Service
	Bookmarks     *bookmarkapp.Service
	Comments      *commentapp.Service
	Reactions     *reactionapp.Service
	Reports       *reportapp.Service
	Moderation    *moderationapp.Service
	Notifications *notificationapp.Service
	Chat          *chatapp.Service
	Media         *mediaapp.Service
}

func New(ctx context.Context, cfg config.Config, log *slog.Logger) (*Container, error) {
	pool, err := db.Open(ctx, cfg.Database)
	if err != nil {
		return nil, err
	}

	txManager := db.NewTxManager(pool)
	authManager := platformauth.NewManager(
		cfg.Auth.Secret,
		cfg.Auth.SessionCookieName,
		cfg.Auth.CSRFCookieName,
		cfg.Auth.CSRFHeaderName,
		cfg.Env.IsProduction(),
	)
	identityService := identityapp.NewService(
		cfg.Auth,
		txManager,
		identitypg.NewUserRepository(txManager),
		identitypg.NewSessionRepository(txManager),
		identitypg.NewEmailCodeRepository(txManager),
		authManager,
		platformoauth.NewHTTPClient(5*time.Second),
	)
	profileService := profileapp.NewService(txManager, profilepg.NewRepository(txManager))
	articleService := articleapp.NewServiceWithSearch(txManager, articlepg.NewRepository(txManager), search.NewArticleSearcher(cfg.Search))
	bookmarkService := bookmarkapp.NewService(txManager, bookmarkpg.NewRepository(txManager))
	commentService := commentapp.NewService(commentpg.NewRepository(txManager))
	reactionService := reactionapp.NewService(txManager, reactionpg.NewRepository(txManager))
	reportService := reportapp.NewService(txManager, reportpg.NewRepository(txManager))
	moderationService := moderationapp.NewService(txManager, moderationpg.NewRepository(txManager))
	notificationService := notificationapp.NewService(notificationpg.NewRepository(txManager))
	chatService := chatapp.NewService(txManager, chatpg.NewRepository(txManager))
	mediaStorage, err := mediastorage.New(ctx, cfg.Storage)
	if err != nil {
		pool.Close()
		return nil, err
	}
	mediaService := mediaapp.NewService(
		mediapg.NewRepository(txManager),
		mediaStorage,
		cfg.Storage.PublicBaseURL,
		cfg.Storage.MaxUploadBytes,
	)

	return &Container{
		Config:        cfg,
		Logger:        log,
		DB:            pool,
		TxManager:     txManager,
		Auth:          authManager,
		Identity:      identityService,
		Profiles:      profileService,
		Articles:      articleService,
		Bookmarks:     bookmarkService,
		Comments:      commentService,
		Reactions:     reactionService,
		Reports:       reportService,
		Moderation:    moderationService,
		Notifications: notificationService,
		Chat:          chatService,
		Media:         mediaService,
	}, nil
}

func (c *Container) Close() {
	if c.DB != nil {
		c.DB.Close()
	}
}
