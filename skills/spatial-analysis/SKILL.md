---
name: spatial-analysis
description: Guide agent through geospatial data analysis tasks using built-in GDAL tools
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
Key things to check:
- Is it geographic (lat/lon in degrees) or projected (in metres/feet)?
- For area calculations, you need a projected CRS (e.g., UTM zone)
- For distance calculations, you need a projected CRS
- EPSG:4326 (WGS 84) is the most common geographic CRS
- EPSG:4490 (CGCS2000) is commonly used in China

### Step 4: Check the Spatial SQL Codebook
When the task maps to a common PostGIS pattern, inspect the codebook first:
```
geo_sql_codebook(action="list", intent="<analysis goal>")
```
Render a verified pattern before writing SQL from scratch:
```
geo_sql_codebook(action="render", pattern="<pattern_name>", values={...})
```

### Step 5: Inspect PostGIS Before Querying
When analysis involves a PostGIS database, inspect the available schema before composing SQL:
```
geo_spatial_query(action="schema")
```
Use the returned tables, columns, geometry types, and SRIDs to write the query.

### Step 6: Execute Spatial SQL
Run the final read-only SQL only after checking the schema:
```
geo_spatial_query(action="query", sql="SELECT ...")
```
Prefer `SELECT`, `WITH`, or `EXPLAIN` queries only.

### Step 7: Process the Data
Use `geo_process` for GDAL operations:

**Reprojection:**
```
geo_process(command="gdalwarp", args=["-t_srs", "EPSG:4326", "input.tif", "output.tif"])
```

**Raster clipping:**
```
geo_process(command="gdalwarp", args=["-cutline", "boundary.shp", "-crop_to_cutline", "input.tif", "clipped.tif"])
```

**Vector conversion with reprojection:**
```
geo_process(command="ogr2ogr", args=["-f", "GeoJSON", "-t_srs", "EPSG:4326", "output.geojson", "input.shp"])
```

### Step 8: Convert Format if Needed
Use `geo_format_convert` for simple format changes:
```
geo_format_convert(input_path="data.shp", output_path="data.geojson")
```

## Key Conventions

- **Discover candidate datasets before assuming they already exist** — use `geo_data_catalog` for local, OSM, or STAC discovery.
- **Always inspect data before processing** — understanding the CRS and extent prevents errors.
- **Always verify CRS compatibility** before spatial operations.
- **Check the SQL codebook before freeform SQL** — prefer `geo_sql_codebook` lookups and rendered patterns first.
- **Inspect PostGIS schema before querying** — use `geo_spatial_query(action="schema")` before `action="query"`.
- **Prefer EPSG:4326** (WGS 84) for output when no specific CRS is requested.
- **Use GeoPackage (.gpkg)** as the default output format for vectors — it's modern and avoids Shapefile limitations.
- **Use GeoTIFF (.tif)** as the default output format for rasters.
- **China-specific**: CGCS2000 (EPSG:4490) is geometrically almost identical to WGS84 but is the official CRS.

## Common Mistakes to Avoid

1. **Don't guess where the data is** — search local, OSM, or STAC sources first.
2. **Don't mix CRS** — always reproject to a common CRS before overlay analysis.
3. **Don't assume WGS84** — always check with `geo_crs_detect`.
4. **Don't use Shapefile for web** — use GeoJSON or GeoPackage instead.
5. **Don't calculate areas in geographic CRS** — reproject to a local projected CRS first.
6. **Don't skip the codebook** — reuse verified patterns before writing custom SQL.
7. **Don't guess PostGIS table shapes** — inspect schema first, then query.
