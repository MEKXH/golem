package geopipeline

import "testing"

func TestBuildReuseCandidates_UsesMatchedPipelineSteps(t *testing.T) {
	matches := []Match{{
		ID:   "pipeline-land-change",
		Goal: "analyze land use change",
		Steps: []Step{
			{Tool: "geo_data_catalog", ArgsJSON: `{"action":"stac_search","collections":["sentinel-2-l2a"]}`},
			{Tool: "geo_process", ArgsJSON: `{"command":"gdalwarp","args":["-t_srs","EPSG:3857"]}`},
		},
		Score: 4,
	}}

	candidates := BuildReuseCandidates(matches)
	if len(candidates) != 1 {
		t.Fatalf("expected one reuse candidate, got %+v", candidates)
	}
	if candidates[0].Goal != "analyze land use change" {
		t.Fatalf("expected matching goal, got %+v", candidates[0])
	}
	if len(candidates[0].Steps) != 2 {
		t.Fatalf("expected two reuse steps, got %+v", candidates[0].Steps)
	}
	if candidates[0].Steps[0].Tool != "geo_data_catalog" || candidates[0].Steps[1].Tool != "geo_process" {
		t.Fatalf("expected original step order to be preserved, got %+v", candidates[0].Steps)
	}
	if !candidates[0].Steps[0].NeedsParameterUpdate || !candidates[0].Steps[1].NeedsParameterUpdate {
		t.Fatalf("expected steps with args_json to require parameter updates, got %+v", candidates[0].Steps)
	}
	if candidates[0].Steps[0].ExampleArgsJSON == "" || candidates[0].Steps[1].ExampleArgsJSON == "" {
		t.Fatalf("expected example args to be surfaced, got %+v", candidates[0].Steps)
	}
}

func TestBuildReuseCandidates_HandlesStepsWithoutArgs(t *testing.T) {
	matches := []Match{{
		ID:   "pipeline-river",
		Goal: "analyze river sinuosity",
		Steps: []Step{
			{Tool: "geo_info", ArgsJSON: ""},
			{Tool: "geo_sinuosity", ArgsJSON: `{"input_path":"river.geojson"}`},
		},
		Score: 2,
	}}

	candidates := BuildReuseCandidates(matches)
	if len(candidates) != 1 {
		t.Fatalf("expected one reuse candidate, got %+v", candidates)
	}
	if candidates[0].Steps[0].NeedsParameterUpdate {
		t.Fatalf("expected arg-less step to remain static, got %+v", candidates[0].Steps[0])
	}
	if candidates[0].Steps[1].ExampleArgsJSON != `{"input_path":"river.geojson"}` {
		t.Fatalf("expected example args to be preserved, got %+v", candidates[0].Steps[1])
	}
}
