---
name: data-pipeline
description: Guide agent through geospatial ETL workflows using built-in, learned, and fabricated geo tools
---

## Data Pipeline Workflow

Use this skill for geospatial ETL tasks: ingest, normalize, convert, reproject, and prepare datasets for analysis.

### Step 1: Discover and Inspect Inputs
Start by locating candidate datasets with `geo_data_catalog`, then inspect them with `geo_info` and `geo_crs_detect`.

### Step 2: Reuse Learned Pipelines First
Before composing a new ETL flow, inspect whether `pipelines/geo/` already contains a similar learned sequence for the same transformation goal.

### Step 3: Normalize CRS and Format
Use `geo_process` when the workflow needs reprojection, clipping, or batch GDAL/OGR steps.
Use `geo_format_convert` for direct format changes.

### Step 4: Reuse Verified SQL Patterns First
When the pipeline targets PostGIS, inspect the codebook before composing custom SQL:
```
geo_sql_codebook(action="list", intent="<pipeline validation goal>")
geo_sql_codebook(action="render", pattern="<pattern_name>", values={...})
geo_spatial_query(action="schema")
geo_spatial_query(action="query", sql="SELECT ...")
```

### Step 5: Fabricate a Reusable Pipeline Tool
If the ETL step is repetitive and not covered by learned pipelines or built-in geo tools, fabricate a persistent workspace tool under `tools/geo/`.

## Conventions

- Prefer learned pipelines before inventing a fresh ETL sequence.
- Prefer GeoPackage for intermediate vector outputs.
- Prefer GeoTIFF for intermediate raster outputs.
- Keep intermediate and final outputs inside the workspace.
