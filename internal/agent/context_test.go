package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MEKXH/golem/internal/skills"
)

func TestBuildSystemPrompt_IncludesRecentDiaries(t *testing.T) {
	workspace := t.TempDir()
	memDir := filepath.Join(workspace, "memory")
	if err := os.MkdirAll(memDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("long-term notes"), 0644); err != nil {
		t.Fatalf("WriteFile MEMORY: %v", err)
	}
	diaries := map[string]string{
		"2026-02-08.md": "oldest",
		"2026-02-09.md": "d2",
		"2026-02-10.md": "d3",
		"2026-02-11.md": "latest",
	}
	for name, content := range diaries {
		if err := os.WriteFile(filepath.Join(memDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	cb := NewContextBuilder(workspace)
	prompt := cb.BuildSystemPrompt()

	if !strings.Contains(prompt, "Long-term Memory") || !strings.Contains(prompt, "long-term notes") {
		t.Fatalf("expected long-term memory in prompt, got: %s", prompt)
	}
	if strings.Contains(prompt, "oldest") {
		t.Fatalf("did not expect oldest diary in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "d2") || !strings.Contains(prompt, "d3") || !strings.Contains(prompt, "latest") {
		t.Fatalf("expected three most recent diaries in prompt, got: %s", prompt)
	}
}

func TestBuildMessages_IncludesMediaList(t *testing.T) {
	cb := NewContextBuilder(t.TempDir())
	msgs := cb.BuildMessages(nil, "analyze this", []string{"a.png", "b.txt"})
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	last := msgs[len(msgs)-1]
	if !strings.Contains(last.Content, "Attached media") {
		t.Fatalf("expected attached media section, got: %s", last.Content)
	}
	if !strings.Contains(last.Content, "a.png") || !strings.Contains(last.Content, "b.txt") {
		t.Fatalf("expected media names included, got: %s", last.Content)
	}
}

func TestBuildSystemPrompt_IncludesBuiltinSkillsSummary(t *testing.T) {
	workspace := t.TempDir()
	builtin := filepath.Join(t.TempDir(), "builtin-skills")
	t.Setenv("GOLEM_BUILTIN_SKILLS_DIR", builtin)

	if err := os.MkdirAll(filepath.Join(builtin, "weather"), 0755); err != nil {
		t.Fatalf("MkdirAll builtin skill: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(builtin, "weather", "SKILL.md"),
		[]byte("---\nname: weather\ndescription: \"builtin weather\"\n---\n\n# Weather\n"),
		0644,
	); err != nil {
		t.Fatalf("WriteFile builtin SKILL.md: %v", err)
	}

	cb := NewContextBuilder(workspace)
	prompt := cb.BuildSystemPrompt()

	if !strings.Contains(prompt, "Installed Skills") {
		t.Fatalf("expected skills section in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "weather") || !strings.Contains(prompt, "builtin weather") {
		t.Fatalf("expected builtin skill summary in prompt, got: %s", prompt)
	}
}

func TestBuildSystemPrompt_IncludesGeoCodebookSummary(t *testing.T) {
	workspace := t.TempDir()
	codebookDir := filepath.Join(workspace, "geo-codebook")
	if err := os.MkdirAll(codebookDir, 0o755); err != nil {
		t.Fatalf("MkdirAll codebook: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(codebookDir, "patterns.yaml"),
		[]byte("name: postgis-core\ndescription: Common patterns\npatterns:\n  - name: point-buffer-count\n    description: Count features in a buffer\n    template: SELECT 1\n    verified: true\n    success_rate: 0.98\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile codebook: %v", err)
	}

	cb := NewContextBuilder(workspace)
	prompt := cb.BuildSystemPrompt()

	if !strings.Contains(prompt, "Spatial SQL Codebook") {
		t.Fatalf("expected codebook summary in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "point-buffer-count") {
		t.Fatalf("expected pattern name in prompt, got: %s", prompt)
	}
}

func TestBuildSystemPrompt_IncludesGeoToolFabricationGuide(t *testing.T) {
	workspace := t.TempDir()
	scriptPath := filepath.Join(workspace, "tools", "geo", "scripts", "sinuosity.py")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("MkdirAll script dir: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("print('ok')\n"), 0o644); err != nil {
		t.Fatalf("WriteFile script: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(workspace, "tools", "geo", "geo_sinuosity.yaml"),
		[]byte("name: geo_sinuosity\ndescription: River sinuosity calculator\nrunner: python\nscript: tools/geo/scripts/sinuosity.py\nparameters:\n  input_path:\n    type: string\n    required: true\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile manifest: %v", err)
	}

	cb := NewContextBuilder(workspace)
	prompt := cb.BuildSystemPrompt()

	if !strings.Contains(prompt, "Geo Tool Fabrication") {
		t.Fatalf("expected geo tool fabrication guide in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "tools/geo/scripts/") || !strings.Contains(prompt, "tools/geo/<tool_name>.yaml") {
		t.Fatalf("expected fabrication file paths in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "The manifest `name` and `<tool_name>` filename must both start with `geo_`") {
		t.Fatalf("expected explicit geo_ naming rule in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "geo_sinuosity") {
		t.Fatalf("expected installed fabricated tool in prompt, got: %s", prompt)
	}
}

func TestBuildSystemPrompt_IncludesLearnedGeoPipelines(t *testing.T) {
	workspace := t.TempDir()
	pipelinesDir := filepath.Join(workspace, "pipelines", "geo")
	if err := os.MkdirAll(pipelinesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll pipelines: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(pipelinesDir, "pipeline-1.yaml"),
		[]byte("id: pipeline-1\ngoal: analyze river sinuosity\ncreated_at: \"2026-03-14T21:30:00Z\"\nsteps:\n  - tool: geo_info\n    args_json: '{\"path\":\"river.geojson\"}'\n  - tool: geo_sinuosity\n    args_json: '{\"input_path\":\"river.geojson\"}'\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile pipeline: %v", err)
	}

	cb := NewContextBuilder(workspace)
	prompt := cb.BuildSystemPrompt()

	if !strings.Contains(prompt, "Learned Geo Pipelines") {
		t.Fatalf("expected learned pipelines section in prompt, got: %s", prompt)
	}
	if !strings.Contains(prompt, "analyze river sinuosity") || !strings.Contains(prompt, "geo_sinuosity") {
		t.Fatalf("expected learned pipeline summary in prompt, got: %s", prompt)
	}
}

func TestBuildMessages_UsesKeywordMemoryRecallWithStats(t *testing.T) {
	workspace := t.TempDir()
	memDir := filepath.Join(workspace, "memory")
	if err := os.MkdirAll(memDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "MEMORY.md"), []byte("payment timeout runbook and mitigation"), 0o644); err != nil {
		t.Fatalf("WriteFile MEMORY: %v", err)
	}
	if err := os.WriteFile(filepath.Join(memDir, "2026-02-15.md"), []byte("- [08:00:00] payment timeout in us-east"), 0o644); err != nil {
		t.Fatalf("WriteFile diary: %v", err)
	}

	cb := NewContextBuilder(workspace)
	msgs := cb.BuildMessages(nil, "Investigate payment timeout", nil)
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(msgs))
	}
	sys := msgs[0].Content
	if !strings.Contains(sys, "Memory Recall") {
		t.Fatalf("expected memory recall section in system prompt, got: %s", sys)
	}
	if !strings.Contains(sys, "recall_count:") || !strings.Contains(sys, "hit_sources:") {
		t.Fatalf("expected recall observability fields in prompt, got: %s", sys)
	}
	if !strings.Contains(strings.ToLower(sys), "payment timeout") {
		t.Fatalf("expected keyword recall content in prompt, got: %s", sys)
	}
}

func TestBuildMessages_IncludesRelevantLearnedGeoPipelinesForInput(t *testing.T) {
	workspace := t.TempDir()
	pipelinesDir := filepath.Join(workspace, "pipelines", "geo")
	if err := os.MkdirAll(pipelinesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll pipelines: %v", err)
	}

	files := map[string]string{
		"pipeline-land-change.yaml": `id: pipeline-land-change
goal: analyze land use change
created_at: "2026-03-14T21:30:00Z"
steps:
  - tool: geo_data_catalog
    args_json: '{}'
  - tool: geo_process
    args_json: '{}'
`,
		"pipeline-river.yaml": `id: pipeline-river
goal: analyze river sinuosity
created_at: "2026-03-13T21:30:00Z"
steps:
  - tool: geo_info
    args_json: '{}'
  - tool: geo_sinuosity
    args_json: '{}'
`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(pipelinesDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error: %v", name, err)
		}
	}

	cb := NewContextBuilder(workspace)
	msgs := cb.BuildMessages(nil, "analyze land use change in a new area", nil)
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(msgs))
	}

	sys := msgs[0].Content
	if !strings.Contains(sys, "Relevant Learned Geo Pipelines") {
		t.Fatalf("expected relevant learned pipelines section in prompt, got: %s", sys)
	}
	if !strings.Contains(sys, "analyze land use change") {
		t.Fatalf("expected matching learned pipeline in prompt, got: %s", sys)
	}
	if strings.Contains(sys, "analyze river sinuosity") {
		t.Fatalf("did not expect unrelated learned pipeline in prompt, got: %s", sys)
	}
}

func TestBuildSystemPrompt_RecordsShownSkillsTelemetry(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "skills", "spatial-analysis"), 0o755); err != nil {
		t.Fatalf("MkdirAll skill: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(workspace, "skills", "spatial-analysis", "SKILL.md"),
		[]byte("---\nname: spatial-analysis\ndescription: \"workspace geo skill\"\n---\n\n# Spatial Analysis\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile skill: %v", err)
	}

	cb := NewContextBuilder(workspace)
	prompt := cb.BuildSystemPrompt()
	if !strings.Contains(prompt, "spatial-analysis") {
		t.Fatalf("expected skills summary in prompt, got: %s", prompt)
	}

	snapshot, err := skills.NewTelemetryRecorder(workspace).Load()
	if err != nil {
		t.Fatalf("Load telemetry error: %v", err)
	}
	if snapshot.Skills["spatial-analysis"].Shown == 0 {
		t.Fatalf("expected shown counter to be recorded, got %+v", snapshot.Skills["spatial-analysis"])
	}
}

func TestBuildMessages_IncludesReplayReadyGeoReuseHints(t *testing.T) {
	workspace := t.TempDir()
	pipelinesDir := filepath.Join(workspace, "pipelines", "geo")
	if err := os.MkdirAll(pipelinesDir, 0o755); err != nil {
		t.Fatalf("MkdirAll pipelines: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(pipelinesDir, "pipeline-land-change.yaml"),
		[]byte("id: pipeline-land-change\ngoal: analyze land use change\ncreated_at: \"2026-03-14T21:30:00Z\"\nsteps:\n  - tool: geo_data_catalog\n    args_json: '{\"action\":\"stac_search\",\"collections\":[\"sentinel-2-l2a\"]}'\n  - tool: geo_process\n    args_json: '{\"command\":\"gdalwarp\",\"args\":[\"-t_srs\",\"EPSG:3857\"]}'\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile pipeline: %v", err)
	}

	cb := NewContextBuilder(workspace)
	msgs := cb.BuildMessages(nil, "analyze land use change in a new area", nil)
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(msgs))
	}

	sys := msgs[0].Content
	if !strings.Contains(sys, "Relevant Learned Geo Pipelines") {
		t.Fatalf("expected relevant learned pipelines section in prompt, got: %s", sys)
	}
	if !strings.Contains(sys, "needs_parameter_update=true") {
		t.Fatalf("expected parameter-aware reuse hint in prompt, got: %s", sys)
	}
	if !strings.Contains(sys, `{"action":"stac_search","collections":["sentinel-2-l2a"]}`) {
		t.Fatalf("expected example args json in prompt, got: %s", sys)
	}
}

func TestBuildMessages_RecordsSelectedSkillTelemetryForExplicitQueryMatch(t *testing.T) {
	workspace := t.TempDir()
	if err := os.MkdirAll(filepath.Join(workspace, "skills", "spatial-analysis"), 0o755); err != nil {
		t.Fatalf("MkdirAll skill: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(workspace, "skills", "spatial-analysis", "SKILL.md"),
		[]byte("---\nname: spatial-analysis\ndescription: \"workspace geo skill\"\n---\n\n# Spatial Analysis\n"),
		0o644,
	); err != nil {
		t.Fatalf("WriteFile skill: %v", err)
	}

	cb := NewContextBuilder(workspace)
	msgs := cb.BuildMessages(nil, "use spatial analysis to inspect this raster", nil)
	if len(msgs) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(msgs))
	}

	sys := msgs[0].Content
	if !strings.Contains(sys, "Relevant Skills") {
		t.Fatalf("expected relevant skills section in system prompt, got: %s", sys)
	}
	if !strings.Contains(sys, "spatial-analysis") {
		t.Fatalf("expected matched skill in relevant section, got: %s", sys)
	}

	snapshot, err := skills.NewTelemetryRecorder(workspace).Load()
	if err != nil {
		t.Fatalf("Load telemetry error: %v", err)
	}
	if snapshot.Skills["spatial-analysis"].Selected == 0 {
		t.Fatalf("expected selected counter to be recorded, got %+v", snapshot.Skills["spatial-analysis"])
	}
}
