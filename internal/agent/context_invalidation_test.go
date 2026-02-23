package agent

import (
	"path/filepath"
	"testing"
)

// TestContextBuilder_InvalidateCache_Selective verifies that InvalidateCache
// only clears the cache when relevant files are modified.
func TestContextBuilder_InvalidateCache_Selective(t *testing.T) {
	// 1. Setup workspace and context builder
	workspace := t.TempDir()
	cb := NewContextBuilder(workspace)

	// 2. Populate cache
	// buildBaseSystemPromptParts populates cachedBaseParts
	_ = cb.buildBaseSystemPromptParts()

	// Verify cache is populated
	cb.mu.RLock()
	if cb.cachedBaseParts == nil {
		cb.mu.RUnlock()
		t.Fatal("expected cachedBaseParts to be populated after buildBaseSystemPromptParts")
	}
	cb.mu.RUnlock()

	// 3. Invalidate with unrelated file (should NOT clear cache)
	unrelatedFile := filepath.Join(workspace, "src", "main.go")
	cb.InvalidateCache(unrelatedFile)

	cb.mu.RLock()
	if cb.cachedBaseParts == nil {
		cb.mu.RUnlock()
		t.Errorf("cache was cleared for unrelated file: %s", unrelatedFile)
	} else {
		cb.mu.RUnlock()
	}

	// 4. Invalidate with related file (IDENTITY.md) (should clear cache)
	identityFile := filepath.Join(workspace, "IDENTITY.md")
	cb.InvalidateCache(identityFile)

	cb.mu.RLock()
	if cb.cachedBaseParts != nil {
		cb.mu.RUnlock()
		t.Errorf("cache was NOT cleared for related file: %s", identityFile)
	} else {
		cb.mu.RUnlock()
	}

	// Repopulate
	_ = cb.buildBaseSystemPromptParts()

	// 5. Invalidate with skill file (should clear cache)
	skillFile := filepath.Join(workspace, "skills", "weather", "SKILL.md")
	cb.InvalidateCache(skillFile)

	cb.mu.RLock()
	if cb.cachedBaseParts != nil {
		cb.mu.RUnlock()
		t.Errorf("cache was NOT cleared for skill file: %s", skillFile)
	} else {
		cb.mu.RUnlock()
	}

	// Repopulate
	_ = cb.buildBaseSystemPromptParts()

	// 6. Invalidate with empty path (force invalidation) (should clear cache)
	cb.InvalidateCache("")

	cb.mu.RLock()
	if cb.cachedBaseParts != nil {
		cb.mu.RUnlock()
		t.Errorf("cache was NOT cleared for empty path (force invalidation)")
	} else {
		cb.mu.RUnlock()
	}
}
