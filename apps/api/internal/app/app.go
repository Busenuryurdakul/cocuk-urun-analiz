package app

import (
	"context"
	"log"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/config"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/cookies"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mail"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
)

type App struct {
	Config config.Config
	Mongo  *mongoclient.Client
	Redis  *redis.Client
	Auth   *auth.Service
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	mongoClient, err := mongoclient.Connect(ctx, cfg.MongoURI)
	if err != nil {
		return nil, err
	}
	if err := mongoClient.EnsureIndexes(ctx); err != nil {
		return nil, err
	}

	redisClient, err := redis.Connect(ctx, cfg.RedisURL)
	if err != nil {
		return nil, err
	}

	db := mongoClient.DB
	users := repository.NewUserRepository(db)
	orgs := repository.NewOrganizationRepository(db)
	members := repository.NewMemberRepository(db)
	devices := repository.NewDeviceRepository(db)
	sessions := repository.NewSessionRepository(db)
	rotated := repository.NewRotatedRefreshRepository(db)
	pending := repository.NewPendingAuthRepository(db)
	emailVerify := repository.NewEmailVerificationRepository(db)
	deviceVerify := repository.NewDeviceVerificationRepository(db)
	mfaSetup := repository.NewMFASetupRepository(db)
	security := repository.NewSecurityEventRepository(db)

	guard := &tenant.Guard{Members: members, Events: security}
	mailer := mail.NewSMTP(cfg.MailSMTPHost, cfg.MailSMTPPort, cfg.MailFrom)

	policy := auth.SecurityPolicy{
		MaxOTPAttempts:       cfg.MaxOTPAttempts,
		LoginMaxAttempts:     cfg.LoginMaxAttempts,
		LoginLockoutDuration: cfg.LoginLockoutDuration,
		AccessTokenTTL:       cfg.AccessTokenTTL,
		RefreshTokenTTL:      cfg.RefreshTokenTTL,
		EmailVerifyTTL:       24 * time.Hour,
		PendingAuthTTL:       10 * time.Minute,
		MFASetupTTL:          30 * time.Minute,
		DeviceVerifyTTL:      15 * time.Minute,
	}

	bruteForce := &auth.BruteForceGuard{
		Redis:    redisClient,
		Policy:   policy,
		Security: security,
	}

	authSvc := &auth.Service{
		Users:        users,
		Orgs:         orgs,
		Members:      members,
		Devices:      devices,
		Sessions:     sessions,
		Rotated:      rotated,
		Pending:      pending,
		EmailVerify:  emailVerify,
		DeviceVerify: deviceVerify,
		MFASetup:     mfaSetup,
		Security:     security,
		Mail:         mailer,
		Tenant:       guard,
		BruteForce:   bruteForce,
		JWT:          auth.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL),
		Policy:       policy,
		WebBaseURL:   cfg.WebBaseURL,
		MFAIssuer:    cfg.MFAIssuer,
	}

	return &App{
		Config: cfg,
		Mongo:  mongoClient,
		Redis:  redisClient,
		Auth:   authSvc,
	}, nil
}

func (a *App) Ready(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := a.Mongo.Ping(ctx); err != nil {
		return err
	}
	return a.Redis.Ping(ctx)
}

func (a *App) CookieOptions() cookies.Options {
	opts := cookies.DefaultOptions()
	opts.Secure = a.Config.CookieSecure
	return opts
}

func (a *App) Shutdown(ctx context.Context) {
	if err := a.Mongo.Disconnect(ctx); err != nil {
		log.Printf("mongodb disconnect: %v", err)
	}
	if err := a.Redis.Close(); err != nil {
		log.Printf("redis close: %v", err)
	}
}
