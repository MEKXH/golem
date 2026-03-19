---
name: spatial-analysis
description: Guide agent through geospatial data analysis tasks using built-in, learned, and fabricated geo tools
---

## Spatial Analysis Workflow

When the user requests geospatial data analysis, follow this structured approach:

### Step 1: Discover Candidate Data
If the required data is not already present, locate candidate datasets first:
```
geo_data_catalog(action="local_scan", path="<workspace path>")
geo_data_catalog(action="overpass_search", bbox=[minLon,minLat,maxLon,maxLat], tags={"amenity":"school"}, limit=10)
geo_data_catalog(action="stac_search", collections=["sentinel-2-l2a"], bbox=[minLon,minLat,maxLon,maxLat], limit=5)
```

### Step 2: Inspect the Data
Use `geo_info` to understand the data before doing anything:
```
geo_info(path="<file_path>")
```
This tells you the format, CRS, extent, and size.

### Step 3: Check the CRS
Use `geo_crs_detect` to verify the coordinate reference system:
```
geo_crs_detect(path="<file_path>")
```

### Step 4: Reuse Learned Pipelines First
Before inventing a new multi-step flow, check whether the workspace already contains a similar learned geo pipeline in `pipelines/geo/`.
Reuse the same tool sequence when the goal is materially similar.

### Step 5: Check the Spatial SQL Codebook
When the task maps to a common PostGIS pattern, inspect the codebook first:
```
geo_sql_codebook(action="list", intent="<analysis goal>")
geo_sql_codebook(action="render", pattern="<pattern_name>", values={...})
```

### Step 6: Inspect PostGIS Before Querying
When analysis involves a PostGIS database, inspect the available schema before composing SQL:
```
geo_spatial_query(action="schema")
geo_spatial_query(action="query", sql="SELECT ...")
```

### Step 7: Process and Convert Data
Use `geo_process` for GDAL/OGR operations and `geo_format_convert` for direct format changes.

### Step 8: Fabricate a Missing Persistent Tool
If the task is recurrent and no learned pipeline, built-in tool, or verified codebook pattern fits, fabricate a workspace geo tool:
- Create the script under `tools/geo/scripts/`.
- Create the manifest under `tools/geo/<tool_name>.yaml`.
- Use a `geo_` tool name.
- The script will receive tool arguments as JSON on stdin.
- The fabricated tool will auto-register on the next agent startup.

## Key Conventions

- Prefer reusing learned pipelines before fabricating a new tool.
- Prefer built-in tools and verified SQL patterns before creating a persistent tool.
- Keep generated outputs inside the workspace.
