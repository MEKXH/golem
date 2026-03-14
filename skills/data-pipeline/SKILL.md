---
name: data-pipeline
description: Guide agent through geospatial ETL workflows using the built-in geo tools
---

## Data Pipeline Workflow

Use this skill for geospatial ETL tasks: ingest, normalize, convert, reproject, and prepare datasets for analysis.

### Step 1: Inspect Every Input
For each source file:
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

### Step 3: Inspect PostGIS Before Loading or Querying
If the pipeline targets PostGIS, first inspect the visible schema:
```
geo_spatial_query(action="schema")
```
Then use read-only SQL to validate the loaded result shape:
```
geo_spatial_query(action="query", sql="SELECT ...")
```

## Conventions

- Prefer GeoPackage for intermediate vector outputs.
- Prefer GeoTIFF for intermediate raster outputs.
- Do not mix coordinate systems in the same overlay step.
- Keep intermediate and final outputs inside the workspace.
