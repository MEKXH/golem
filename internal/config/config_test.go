package config

import "testing"

func TestDefaultConfig(t *testing.T) {
    cfg := DefaultConfig()

    if cfg.Agents.Defaults.MaxToolIterations != 20 {
        t.Errorf("expected MaxToolIterations=20, got %d", cfg.Agents.Defaults.MaxToolIterations)
    }
    if cfg.Agents.Defaults.Temperature != 0.7 {
        t.Errorf("expected Temperature=0.7, got %f", cfg.Agents.Defaults.Temperature)
    }
    if cfg.Gateway.Port != 18790 {
        t.Errorf("expected Port=18790, got %d", cfg.Gateway.Port)
    }
}
