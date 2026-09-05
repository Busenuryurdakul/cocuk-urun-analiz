package app

import (
	"context"
	"log"
	"time"

	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/agent"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/auth"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/compliance"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/config"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/cookies"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/dataset"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mail"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/marketplace"
	mongoclient "github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/mongo"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/org"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/product"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/queue"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/redis"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/repository"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/storage"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/tenant"
	"github.com/Busenuryurdakul/cocuk-urun-analiz/apps/api/internal/ugc"
)

type App struct {
	Config         config.Config
	Mongo          *mongoclient.Client
	Redis          *redis.Client
	Auth           *auth.Service
	Org            *org.Service
	Compliance     *compliance.Engine
	Consent        *compliance.ConsentService
	PolicyRepo     *compliance.PolicyRepository
	Products       *product.Service
	UGC            *ugc.Service
	Marketplace    *marketplace.Service
	Dataset        *dataset.Service
	Agent          *agent.Service
	ImportConsumer *marketplace.Consumer
	AgentInternal  *agent.InternalHandler
	RecoveryWorker *agent.RecoveryWorker
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
	invitations := repository.NewInvitationRepository(db)
	compliancePolicies := repository.NewCompliancePolicyRepository(db)
	consents := repository.NewConsentRepository(db)
	complianceEvents := repository.NewComplianceEventRepository(db)
	configAudit := repository.NewConfigAuditRepository(db)

	productsRepo := repository.NewProductRepository(db)
	mappingsRepo := repository.NewProductSourceMappingRepository(db)
	uxRepo := repository.NewUserExperienceRepository(db)
	reviewsRepo := repository.NewMarketplaceReviewRepository(db)
	importRunsRepo := repository.NewMarketplaceImportRunRepository(db)
	rawPayloadRepo := repository.NewRawSourcePayloadRepository(db)
	datasetRecordsRepo := repository.NewDatasetRecordRepository(db)
	datasetVersionsRepo := repository.NewDatasetVersionRepository(db)
	analysisRunsRepo := repository.NewAnalysisRunRepository(db)
	agentEventsRepo := repository.NewAgentRunEventRepository(db)
	toolExecutionsRepo := repository.NewToolExecutionRepository(db)
	configSnapshotsRepo := repository.NewConfigSnapshotRepository(db)

	if err := compliance.SeedPlatformPolicies(ctx, compliancePolicies); err != nil {
		return nil, err
	}

	policyRepo := &compliance.PolicyRepository{Policies: compliancePolicies}
	consentSvc := &compliance.ConsentService{
		Consents:   consents,
		Events:     complianceEvents,
		Security:   security,
		PolicyRepo: policyRepo,
		Orgs:       orgs,
	}
	complianceEngine := &compliance.Engine{
		PolicyRepo: policyRepo,
		Consent:    consentSvc,
		Events:     complianceEvents,
		Security:   security,
		Orgs:       orgs,
	}

	guard := &tenant.Guard{Members: members, Events: security}
	mailer := mail.NewSMTP(cfg.MailSMTPHost, cfg.MailSMTPPort, cfg.MailFrom)

	orgSvc := &org.Service{
		Orgs:        orgs,
		Members:     members,
		Users:       users,
		Invitations: invitations,
		Security:    security,
		ConfigAudit: configAudit,
		Policies:    compliancePolicies,
		Mail:        mailer,
		Tenant:      guard,
		Compliance:  complianceEngine,
		Consent:     consentSvc,
		PolicyRepo:  policyRepo,
		WebBaseURL:  cfg.WebBaseURL,
	}

	productSvc := &product.Service{
		Products: productsRepo,
		Mappings: mappingsRepo,
		Security: security,
		Tenant:   guard,
	}

	ugcSvc := &ugc.Service{
		Experiences: uxRepo,
		Products:    productsRepo,
		Consent:     consentSvc,
		Security:    security,
		Tenant:      guard,
	}

	datasetSvc := &dataset.Service{
		Records:  datasetRecordsRepo,
		Versions: datasetVersionsRepo,
		Security: security,
		Tenant:   guard,
	}

	var s3Client *storage.S3Client
	if cfg.S3Endpoint != "" && cfg.S3Bucket != "" {
		s3Client, err = storage.NewS3Client(cfg)
		if err != nil {
			log.Printf("s3 client disabled: %v", err)
		}
	}

	importQueue := &queue.ImportQueue{Redis: redisClient}
	marketplaceSvc := &marketplace.Service{
		Runs:           importRunsRepo,
		Reviews:        reviewsRepo,
		Raw:            rawPayloadRepo,
		Products:       productSvc,
		DatasetRecords: datasetRecordsRepo,
		Registry:       marketplace.DefaultRegistry(),
		Queue:          importQueue,
		Storage:        s3Client,
		Security:       security,
		Tenant:         guard,
	}
	importConsumer := marketplace.NewConsumer(marketplaceSvc, 2*time.Second)

	agentRegistry := agent.NewRegistry()
	agentResolver := &agent.InputResolver{
		Products:   productsRepo,
		Reviews:    reviewsRepo,
		UX:         uxRepo,
		ImportRuns: importRunsRepo,
	}
	agentExecutor := &agent.Executor{Registry: agentRegistry, Compliance: complianceEngine}
	agentAuthorizer := &agent.Authorizer{Registry: agentRegistry, Security: security}
	grantStore := &agent.GrantStore{Redis: redisClient, Security: security, TTL: 5 * time.Minute}
	leaseStore := &agent.LeaseStore{Redis: redisClient, TTL: 2 * time.Minute}
	orchestratorClient := agent.NewOrchestratorClient(cfg.AgentOrchestratorURL, cfg.AgentInternalToken, cfg.AgentIPCTimeout)
	agentSvc := agent.NewService(agent.ServiceDeps{
		Runs:         analysisRunsRepo,
		Events:       agentEventsRepo,
		Executions:   toolExecutionsRepo,
		Snapshots:    configSnapshotsRepo,
		Orgs:         orgs,
		Resolver:     agentResolver,
		Executor:     agentExecutor,
		Authorizer:   agentAuthorizer,
		Compliance:   complianceEngine,
		Registry:     agentRegistry,
		Orchestrator: orchestratorClient,
		Grants:       grantStore,
		Leases:       leaseStore,
		Security:     security,
		Tenant:       guard,
	})
	recoveryWorker := agent.NewRecoveryWorker(agentSvc, leaseStore, 30*time.Second)

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
		Config:         cfg,
		Mongo:          mongoClient,
		Redis:          redisClient,
		Auth:           authSvc,
		Org:            orgSvc,
		Compliance:     complianceEngine,
		Consent:        consentSvc,
		PolicyRepo:     policyRepo,
		Products:       productSvc,
		UGC:            ugcSvc,
		Marketplace:    marketplaceSvc,
		Dataset:        datasetSvc,
		Agent:          agentSvc,
		ImportConsumer: importConsumer,
		AgentInternal: &agent.InternalHandler{
			Service: agentSvc,
			Token:   cfg.AgentInternalToken,
		},
		RecoveryWorker: recoveryWorker,
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

func (a *App) StartBackgroundWorkers() {
	if a.ImportConsumer != nil {
		a.ImportConsumer.Start()
	}
	if a.RecoveryWorker != nil {
		a.RecoveryWorker.Start()
	}
}

func (a *App) Shutdown(ctx context.Context) {
	if a.ImportConsumer != nil {
		a.ImportConsumer.Stop(ctx)
	}
	if a.RecoveryWorker != nil {
		a.RecoveryWorker.Stop()
	}
	if err := a.Mongo.Disconnect(ctx); err != nil {
		log.Printf("mongodb disconnect: %v", err)
	}
	if err := a.Redis.Close(); err != nil {
		log.Printf("redis close: %v", err)
	}
}
