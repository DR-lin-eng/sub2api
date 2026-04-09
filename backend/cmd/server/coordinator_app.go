//go:build !wireinject

package main

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type CoordinatorApplication struct {
	Cleanup func()
}

func initializeCoordinatorApplication(buildInfo handler.BuildInfo) (*CoordinatorApplication, error) {
	_ = buildInfo

	cfg, err := config.ProvideConfig()
	if err != nil {
		return nil, err
	}

	client, err := repository.ProvideEnt(cfg)
	if err != nil {
		return nil, err
	}

	db, err := repository.ProvideSQLDB(client)
	if err != nil {
		return nil, err
	}

	redisClient := repository.ProvideRedis(cfg)
	schedulerCache := repository.NewSchedulerCache(redisClient)
	gatewayCache := repository.NewGatewayCache(redisClient)

	userRepository := repository.NewUserRepository(client, db)
	groupRepository := repository.NewGroupRepository(client, db)
	accountRepository := repository.NewAccountRepository(client, db, schedulerCache)
	soraAccountRepository := repository.NewSoraAccountRepository(db)
	proxyRepository := repository.NewProxyRepository(client, db)
	settingRepository := repository.NewSettingRepository(client)
	usageLogRepository := repository.NewUsageLogRepository(client, db)
	usageBillingRepository := repository.NewUsageBillingRepository(client, db)
	userSubscriptionRepository := repository.NewUserSubscriptionRepository(client)
	apiKeyRepository := repository.NewAPIKeyRepository(client, db)
	userGroupRateRepository := repository.NewUserGroupRateRepository(db)
	opsRepository := repository.NewOpsRepository(db)
	dashboardAggregationRepository := repository.NewDashboardAggregationRepository(db)
	usageCleanupRepository := repository.NewUsageCleanupRepository(client, db)
	scheduledTestPlanRepository := repository.NewScheduledTestPlanRepository(db)
	scheduledTestResultRepository := repository.NewScheduledTestResultRepository(db)
	proxyMaintenancePlanRepository := repository.NewProxyMaintenancePlanRepository(db)
	proxyMaintenanceResultRepository := repository.NewProxyMaintenanceResultRepository(db)
	idempotencyRepository := repository.NewIdempotencyRepository(client, db)

	settingService := service.ProvideSettingService(settingRepository, groupRepository, cfg)
	emailService := service.NewEmailService(settingRepository, repository.NewEmailCache(redisClient))
	emailQueueService := service.ProvideEmailQueueService(emailService)
	billingCache := repository.NewBillingCache(redisClient)
	billingCacheService := service.NewBillingCacheService(billingCache, userRepository, userSubscriptionRepository, apiKeyRepository, cfg)
	subscriptionService := service.NewSubscriptionService(groupRepository, userSubscriptionRepository, billingCacheService, client, cfg)
	userService := service.NewUserService(userRepository, nil, billingCache)

	concurrencyCache := repository.ProvideConcurrencyCache(redisClient, cfg)
	concurrencyService := service.ProvideConcurrencyService(concurrencyCache, accountRepository, cfg)
	timingWheelService, err := service.ProvideTimingWheelService()
	if err != nil {
		return nil, err
	}
	userMessageQueueService := service.ProvideUserMessageQueueService(
		repository.NewUserMsgQueueCache(redisClient),
		repository.NewRPMCache(redisClient),
		cfg,
	)
	_ = userMessageQueueService

	geminiQuotaService := service.NewGeminiQuotaService(cfg, settingRepository)
	tempUnschedCache := repository.NewTempUnschedCache(redisClient)
	timeoutCounterCache := repository.NewTimeoutCounterCache(redisClient)
	geminiTokenCache := repository.NewGeminiTokenCache(redisClient)
	oauthRefreshAPI := service.NewOAuthRefreshAPI(accountRepository, geminiTokenCache)
	compositeTokenCacheInvalidator := service.NewCompositeTokenCacheInvalidator(geminiTokenCache)

	oauthService := service.NewOAuthService(proxyRepository, repository.NewClaudeOAuthClient())
	openAIOAuthService := service.NewOpenAIOAuthService(proxyRepository, repository.NewOpenAIOAuthClient())
	geminiOAuthService := service.NewGeminiOAuthService(
		proxyRepository,
		repository.NewGeminiOAuthClient(cfg),
		repository.NewGeminiCliCodeAssistClient(),
		repository.NewGeminiDriveClient(),
		cfg,
	)
	antigravityOAuthService := service.NewAntigravityOAuthService(proxyRepository)

	rateLimitService := service.ProvideRateLimitService(
		accountRepository,
		usageLogRepository,
		cfg,
		geminiQuotaService,
		tempUnschedCache,
		timeoutCounterCache,
		settingService,
		compositeTokenCacheInvalidator,
	)

	httpUpstream := repository.NewHTTPUpstream(cfg)
	schedulerSnapshotService := service.ProvideSchedulerSnapshotService(
		schedulerCache,
		repository.NewSchedulerOutboxRepository(db),
		accountRepository,
		groupRepository,
		cfg,
	)
	accountImportAccountStore := repository.ProvideAccountImportAccountStore(client, db, schedulerCache)
	accountImportBatchRepository := repository.ProvideAccountImportBatchRepository(redisClient)
	accountImportService := service.ProvideAccountImportService(accountImportAccountStore, accountImportBatchRepository, proxyRepository, groupRepository, soraAccountRepository, schedulerSnapshotService, cfg)
	adminhandler.SetDefaultAccountImportService(accountImportService)

	antigravityTokenProvider := service.ProvideAntigravityTokenProvider(accountRepository, geminiTokenCache, antigravityOAuthService, oauthRefreshAPI, tempUnschedCache)
	antigravityGatewayService := service.NewAntigravityGatewayService(accountRepository, gatewayCache, schedulerSnapshotService, antigravityTokenProvider, rateLimitService, httpUpstream, settingService)
	geminiTokenProvider := service.ProvideGeminiTokenProvider(accountRepository, geminiTokenCache, geminiOAuthService, oauthRefreshAPI)
	kiroUsageService := service.NewKiroUsageService()
	kiroTokenProvider := service.ProvideKiroTokenProvider(accountRepository, geminiTokenCache, kiroUsageService, oauthRefreshAPI)
	kiroGatewayService := service.NewKiroGatewayService(httpUpstream, kiroTokenProvider, kiroUsageService)
	accountTestService := service.NewAccountTestService(accountRepository, geminiTokenProvider, antigravityGatewayService, httpUpstream, cfg)

	pricingService, err := service.ProvidePricingService(cfg, repository.ProvidePricingRemoteClient(cfg))
	if err != nil {
		return nil, err
	}
	billingService := service.NewBillingService(cfg, pricingService)
	deferredService := service.ProvideDeferredService(accountRepository, timingWheelService)
	identityService := service.NewIdentityService(repository.NewIdentityCache(redisClient))
	claudeTokenProvider := service.ProvideClaudeTokenProvider(accountRepository, geminiTokenCache, oauthService, oauthRefreshAPI)
	sessionLimitCache := repository.ProvideSessionLimitCache(redisClient, cfg)
	rpmCache := repository.NewRPMCache(redisClient)
	proxyLatencyCache := repository.NewProxyLatencyCache(redisClient)
	digestStore := service.NewDigestSessionStore()
	adminService := service.NewAdminService(
		userRepository,
		groupRepository,
		accountRepository,
		soraAccountRepository,
		proxyRepository,
		apiKeyRepository,
		repository.NewRedeemCodeRepository(client),
		userGroupRateRepository,
		billingCacheService,
		repository.NewProxyExitInfoProber(cfg),
		proxyLatencyCache,
		nil,
		client,
		settingService,
		subscriptionService,
		userSubscriptionRepository,
		providePrivacyClientFactory(),
	)

	gatewayService := service.ProvideGatewayService(
		accountRepository,
		groupRepository,
		usageLogRepository,
		usageBillingRepository,
		userRepository,
		userSubscriptionRepository,
		userGroupRateRepository,
		gatewayCache,
		cfg,
		schedulerSnapshotService,
		concurrencyService,
		billingService,
		rateLimitService,
		billingCacheService,
		identityService,
		httpUpstream,
		deferredService,
		claudeTokenProvider,
		sessionLimitCache,
		rpmCache,
		digestStore,
		settingService,
		proxyRepository,
		proxyLatencyCache,
		kiroTokenProvider,
		kiroGatewayService,
	)

	openAITokenProvider := service.ProvideOpenAITokenProvider(accountRepository, geminiTokenCache, openAIOAuthService, oauthRefreshAPI)
	openAIGatewayService := service.NewOpenAIGatewayService(
		accountRepository,
		groupRepository,
		usageLogRepository,
		usageBillingRepository,
		userRepository,
		userSubscriptionRepository,
		userGroupRateRepository,
		gatewayCache,
		cfg,
		schedulerSnapshotService,
		concurrencyService,
		billingService,
		rateLimitService,
		billingCacheService,
		httpUpstream,
		deferredService,
		openAITokenProvider,
	)
	openAIGatewayService.SetIdentityService(identityService)

	geminiMessagesCompatService := service.NewGeminiMessagesCompatService(
		accountRepository,
		groupRepository,
		gatewayCache,
		schedulerSnapshotService,
		geminiTokenProvider,
		rateLimitService,
		httpUpstream,
		antigravityGatewayService,
		cfg,
	)

	opsSystemLogSink := service.ProvideOpsSystemLogSink(opsRepository)
	opsService := service.NewOpsService(
		opsRepository,
		settingRepository,
		cfg,
		accountRepository,
		userRepository,
		concurrencyService,
		gatewayService,
		openAIGatewayService,
		geminiMessagesCompatService,
		antigravityGatewayService,
		opsSystemLogSink,
	)

	opsMetricsCollector := service.ProvideOpsMetricsCollector(opsRepository, settingRepository, accountRepository, concurrencyService, db, redisClient, cfg)
	opsAggregationService := service.ProvideOpsAggregationService(opsRepository, settingRepository, db, redisClient, cfg)
	opsAlertEvaluatorService := service.ProvideOpsAlertEvaluatorService(opsService, opsRepository, emailService, redisClient, cfg)
	opsCleanupService := service.ProvideOpsCleanupService(opsRepository, db, redisClient, cfg)
	opsScheduledReportService := service.ProvideOpsScheduledReportService(opsService, userService, emailService, redisClient, cfg)

	dashboardAggregationService := service.ProvideDashboardAggregationService(dashboardAggregationRepository, timingWheelService, cfg)
	usageCleanupService := service.ProvideUsageCleanupService(usageCleanupRepository, timingWheelService, dashboardAggregationService, cfg)
	tokenRefreshService := service.ProvideTokenRefreshService(
		accountRepository,
		soraAccountRepository,
		oauthService,
		openAIOAuthService,
		geminiOAuthService,
		antigravityOAuthService,
		compositeTokenCacheInvalidator,
		schedulerCache,
		cfg,
		tempUnschedCache,
		providePrivacyClientFactory(),
		proxyRepository,
		oauthRefreshAPI,
		rateLimitService,
	)
	schedulerSnapshotService = service.ProvideSchedulerSnapshotAdmissionBinding(schedulerSnapshotService, tokenRefreshService)
	accountExpiryService := service.ProvideAccountExpiryService(accountRepository)
	accountModelsRefreshService := service.ProvideAccountModelsRefreshService(accountRepository, accountTestService)
	subscriptionExpiryService := service.ProvideSubscriptionExpiryService(userSubscriptionRepository)
	scheduledTestService := service.ProvideScheduledTestService(scheduledTestPlanRepository, scheduledTestResultRepository)
	scheduledTestRunnerService := service.ProvideScheduledTestRunnerService(scheduledTestPlanRepository, scheduledTestService, accountTestService, rateLimitService, cfg)
	proxyMaintenanceService := service.ProvideProxyMaintenanceService(proxyMaintenancePlanRepository, proxyMaintenanceResultRepository, adminService, settingService)
	proxyMaintenanceRunnerService := service.ProvideProxyMaintenanceRunnerService(proxyMaintenanceService, cfg)
	soraMediaStorage := service.ProvideSoraMediaStorage(cfg)
	soraMediaCleanupService := service.ProvideSoraMediaCleanupService(soraMediaStorage, cfg)
	idempotencyCleanupService := service.ProvideIdempotencyCleanupService(idempotencyRepository, cfg)

	secretEncryptor, err := repository.NewAESEncryptor(cfg)
	if err != nil {
		return nil, err
	}
	backupService := service.ProvideBackupService(
		settingRepository,
		cfg,
		secretEncryptor,
		repository.NewS3BackupStoreFactory(),
		repository.NewPgDumper(cfg),
	)

	cleanup := provideCleanup(
		client,
		redisClient,
		opsMetricsCollector,
		opsAggregationService,
		opsAlertEvaluatorService,
		opsCleanupService,
		opsScheduledReportService,
		opsSystemLogSink,
		soraMediaCleanupService,
		schedulerSnapshotService,
		tokenRefreshService,
		accountExpiryService,
		accountModelsRefreshService,
		subscriptionExpiryService,
		usageCleanupService,
		idempotencyCleanupService,
		pricingService,
		emailQueueService,
		billingCacheService,
		nil,
		subscriptionService,
		oauthService,
		openAIOAuthService,
		geminiOAuthService,
		antigravityOAuthService,
		nil,
		openAIGatewayService,
		scheduledTestRunnerService,
		proxyMaintenanceRunnerService,
		backupService,
	)

	return &CoordinatorApplication{Cleanup: cleanup}, nil
}
