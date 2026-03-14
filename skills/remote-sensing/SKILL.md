---
name: remote-sensing
description: Guide agent through raster and satellite imagery analysis with the built-in, learned, and fabricated geo tools
---

## Remote Sensing Workflow

Use this skill when the task involves satellite imagery, raster products, vegetation indices, terrain, or image preprocessing.

### Step 1: Discover Candidate Imagery
If the user does not already provide raster files, discover likely datasets first with `geo_data_catalog`.

### Step 2: Inspect and Verify CRS
Start with `geo_info` and `geo_crs_detect`.

### Step 3: Reuse Learned Pipelines First
Before creating a new raster workflow, inspect whether `pipelines/geo/` already captures a similar scene-prep or raster-analysis sequence.

### Step 4: Use GDAL for Raster Processing
Typical operations go through `geo_process`.

### Step 5: Convert Deliverables
Use `geo_format_convert` when the result needs a simpler output format.

### Step 6: Fabricate a Reusable Raster Tool
If a raster workflow is clearly reusable and not covered by learned pipelines or built-in tools, fabricate a workspace geo tool under `tools/geo/`.

## Conventions

- Prefer learned pipelines before fabricating a new raster tool.
- Prefer GeoTIFF for raster outputs unless the user asks for a web-ready preview.
- Keep all generated outputs inside the workspace.
