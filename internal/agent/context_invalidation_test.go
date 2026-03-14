package agent

import (
	"path/filepath"
	"testing"
)

// TestContextBuilder_InvalidateCache_Selective verifies that InvalidateCache
// only clears the cache when relevant files are modified.
func TestContextBuilder_InvalidateCache_Selective(t *testing.T) {
	workspace := t.TempDir()
	cb := NewContextBuilder(workspace)

	_ = cb.buildBaseSystemPromptParts()

	cb.mu.RLock()
	if cb.cachedBaseParts == nil {
		cb.mu.RUnlock()
		t.Fatal("expected cachedBaseParts to be populated after buildBaseSystemPromptParts")
	}
	cb.mu.RUnlock()

	unrelatedFile := filepath.Join(workspace, "src", "main.go")
	cb.InvalidateCache(unrelatedFile)

	cb.mu.RLock()
	if cb.cachedBaseParts == nil {
		cb.mu.RUnlock()
		t.Errorf("cache was cleared for unrelated file: %s", unrelatedFile)
	} else {
		cb.mu.RUnlock()
	}

	identityFile := filepath.Join(workspace, "IDENTITY.md")
	cb.InvalidateCache(identityFile)

	cb.mu.RLock()
	if cb.cachedBaseParts != nil {
		cb.mu.RUnlock()
		t.Errorf("cache was NOT cleared for related file: %s", identityFile)
	} else {
		cb.mu.RUnlock()
	}

	_ = cb.buildBaseSystemPromptParts()

	skillFile := filepath.Join(workspace, "skills", "weather", "SKILL.md")
	cb.InvalidateCache(skillFile)

	cb.mu.RLock()
	if cb.cachedBaseParts != nil {
		cb.mu.RUnlock()
		t.Errorf("cache was NOT cleared for skill file: %s", skillFile)
	} else {
		cb.mu.RUnlock()
	}

	_ = cb.buildBaseSystemPromptParts()

	codebookFile := filepath.Join(workspace, "geo-codebook", "postgis-core.yaml")
	cb.InvalidateCache(codebookFile)

	cb.mu.RLock()
	if cb.cachedBaseParts != nil {
		cb.mu.RUnlock()
		t.Errorf("cache was NOT cleared for codebook file: %s", codebookFile)
	} else {
		cb.mu.RUnlock()
	}

	_ = cb.buildBaseSystemPromptParts()

	cb.InvalidateCache("")

	cb.mu.RLock()
	if cb.cachedBaseParts != nil {
		cb.mu.RUnlock()
		t.Errorf("cache was NOT cleared for empty path (force invalidation)")
	} else {
		cb.mu.RUnlock()
	}
}
