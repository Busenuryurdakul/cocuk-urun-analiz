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
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
)

type App struct {
	Config config.Config
	Mongo  *mongoclient.Client
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

	db := mongoClient.DB
	users := repository.NewUserRepository(db)
	orgs := repository.NewOrganizationRepository(db)
	members := repository.NewMemberRepository(db)
	devices := repository.NewDeviceRepository(db)
	sessions := repository.NewSessionRepository(db)
	pending := repository.NewPendingAuthRepository(db)
	emailVerify := repository.NewEmailVerificationRepository(db)
	deviceVerify := repository.NewDeviceVerificationRepository(db)
	mfaSetup := repository.NewMFASetupRepository(db)
	security := repository.NewSecurityEventRepository(db)

	guard := &tenant.Guard{Members: members, Events: security}
	mailer := mail.NewSMTP(cfg.MailSMTPHost, cfg.MailSMTPPort, cfg.MailFrom)

	authSvc := &auth.Service{
		Users:        users,
		Orgs:         orgs,
		Members:      members,
		Devices:      devices,
		Sessions:     sessions,
		Pending:      pending,
		EmailVerify:  emailVerify,
		DeviceVerify: deviceVerify,
		MFASetup:     mfaSetup,
		Security:     security,
		Mail:         mailer,
		Tenant:       guard,
		WebBaseURL:   cfg.WebBaseURL,
		MFAIssuer:    cfg.MFAIssuer,
	}

	return &App{
		Config: cfg,
		Mongo:  mongoClient,
		Auth:   authSvc,
	}, nil
}

func (a *App) Ready(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return a.Mongo.Ping(ctx)
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
}
