package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	auditpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/audit"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/config"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/database"
	delegpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/delegation"
	fedpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/federation"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/handler"
	"github.com/YASSERRMD/unified-trust-platform/backend/internal/http/router"
	jitpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/jit"
	mfapkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/mfa"
	policypkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/policy"
	tenantpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/tenant"
	userpkg "github.com/YASSERRMD/unified-trust-platform/backend/internal/user"
	"github.com/redis/go-redis/v9"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	ctx := context.Background()

	db, err := database.NewPool(ctx, cfg.Database)
	if err != nil {
		logger.Warn("database unavailable at startup", zap.Error(err))
	} else {
		logger.Info("database connected")
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Warn("redis unavailable at startup", zap.Error(err))
	} else {
		logger.Info("redis connected")
	}

	handlers := &router.Handlers{
		Health: handler.NewHealthHandler(db, rdb),
	}

	if db != nil {
		handlers.Tenant = tenantpkg.NewHandler(tenantpkg.NewService(db))
		handlers.User = userpkg.NewHandler(userpkg.NewService(db))

		mfaSvc := mfapkg.NewService(db, cfg.JWT.Issuer)
		handlers.MFA = mfapkg.NewHandler(mfaSvc)

		rbacSvc := policypkg.NewRBACService(db)
		handlers.RBAC = policypkg.NewRBACHandler(rbacSvc)

		engine := policypkg.NewPolicyEngine(db, rbacSvc)
		handlers.Policy = policypkg.NewPolicyHandler(engine)

		handlers.Delegation = delegpkg.NewHandler(delegpkg.NewService(db))
		handlers.JIT = jitpkg.NewHandler(jitpkg.NewService(db))
		handlers.Federation = fedpkg.NewHandler(fedpkg.NewService(db))
		handlers.Audit = auditpkg.NewHandler(auditpkg.NewService(db))
	}

	h := router.New(logger, rdb, handlers)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      h,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		logger.Info("server starting", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	if db != nil {
		db.Close()
	}
	if err := rdb.Close(); err != nil {
		logger.Warn("redis close error", zap.Error(err))
	}

	logger.Info("server stopped")
}
