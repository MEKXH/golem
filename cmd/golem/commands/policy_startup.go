package commands

import (
	"context"
	"log/slog"
	"strings"

	"github.com/MEKXH/golem/internal/agent"
	"github.com/MEKXH/golem/internal/config"
)

func persistentOffWarningMessage(mode, offTTL string) (string, bool) {
	if strings.EqualFold(strings.TrimSpace(mode), "off") && strings.TrimSpace(offTTL) == "" {
		return "HIGH-RISK policy mode detected: policy.mode=off without policy.off_ttl leaves all guarded tools unrestricted indefinitely", true
	}
	return "", false
}

func logAndAuditRuntimePolicyStartup(ctx context.Context, loop *agent.Loop, cfg *config.Config) {
	if cfg == nil {
		return
	}

	slog.Info("runtime policy configured",
		"mode", cfg.Policy.Mode,
		"off_ttl", cfg.Policy.OffTTL,
		"allow_persistent_off", cfg.Policy.AllowPersistentOff,
		"require_approval", cfg.Policy.RequireApproval,
	)

	if warning, ok := persistentOffWarningMessage(cfg.Policy.Mode, cfg.Policy.OffTTL); ok {
		slog.Warn(warning,
			"mode", "off",
			"off_ttl", "none",
			"allow_persistent_off", cfg.Policy.AllowPersistentOff,
			"mitigation", "set policy.off_ttl or switch policy.mode to strict/relaxed",
		)
	}

	if loop != nil {
		loop.AuditRuntimePolicyStartup(ctx, cfg)
	}
}
