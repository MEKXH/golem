---
name: data-pipeline
description: Guide agent through geospatial ETL workflows using built-in and fabricated geo tools
---

## Data Pipeline Workflow

Use this skill for geospatial ETL tasks: ingest, normalize, convert, reproject, and prepare datasets for analysis.

### Step 1: Discover and Inspect Inputs
Start by locating candidate datasets:
```
geo_data_catalog(action="local_scan", path="<workspace path>")
geo_data_catalog(action="overpass_search", bbox=[minLon,minLat,maxLon,maxLat], tags={"amenity":"school"}, limit=10)
geo_data_catalog(action="stac_search", collections=["sentinel-2-l2a"], bbox=[minLon,minLat,maxLon,maxLat], limit=5)
```
Then inspect each selected file:
```
geo_info(path="<input_path>")
geo_crs_detect(path="<input_path>")
```

### Step 2: Normalize CRS and Format
Use `geo_process` when the workflow needs reprojection, clipping, or batch GDAL/OGR steps:
```
geo_process(command="ogr2ogr", args=["-f", "GPKG", "-t_srs", "EPSG:4326", "output.gpkg", "input.shp"])
geo_process(command="gdalwarp", args=["-t_srs", "EPSG:4326", "input.tif", "normalized.tif"])
```

Use `geo_format_convert` for direct format changes:
```
geo_format_convert(input_path="roads.shp", output_path="roads.geojson")
```

### Step 3: Reuse Verified SQL Patterns First
When the pipeline targets PostGIS, inspect the codebook before composing custom SQL:
```
geo_sql_codebook(action="list", intent="<pipeline validation goal>")
geo_sql_codebook(action="render", pattern="<pattern_name>", values={...})
```

### Step 4: Inspect PostGIS Before Loading or Querying
If the pipeline targets PostGIS, first inspect the visible schema:
```
geo_spatial_query(action="schema")
```
Then use read-only SQL to validate the loaded result shape:
```
geo_spatial_query(action="query", sql="SELECT ...")
```

### Step 5: Fabricate a Reusable Pipeline Tool
If the ETL step is repetitive and not covered by the built-in geo tools, fabricate a persistent workspace tool:
- Put the executable script under `tools/geo/scripts/`.
- Put the manifest under `tools/geo/<tool_name>.yaml`.
- Use a `geo_` tool name and describe the expected parameters explicitly.
- Fabricated tool input arrives as JSON on stdin.
- The tool will register automatically on the next agent startup.

## Conventions

- Prefer GeoPackage for intermediate vector outputs.
- Prefer GeoTIFF for intermediate raster outputs.
- Reuse verified codebook patterns before writing ad-hoc SQL.
- Do not mix coordinate systems in the same overlay step.
- Keep intermediate and final outputs inside the workspace.
- Prefer fabricating reusable ETL helpers only after confirming the built-in geo tools are insufficient.
