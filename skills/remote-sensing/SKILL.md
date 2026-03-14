---
name: remote-sensing
description: Guide agent through raster and satellite imagery analysis with the built-in geo tools
---

## Remote Sensing Workflow

Use this skill when the task involves satellite imagery, raster products, vegetation indices, terrain, or image preprocessing.

### Step 1: Inspect the Raster
Start with:
```
geo_info(path="<raster_path>")
```

### Step 2: Verify CRS Before Raster Math
Check whether the raster is in the right coordinate system:
```
geo_crs_detect(path="<raster_path>")
```
Reproject first when the downstream analysis requires a projected CRS.

### Step 3: Use GDAL for Raster Processing
Typical operations go through `geo_process`:
```
geo_process(command="gdalwarp", args=["-t_srs", "EPSG:4326", "input.tif", "output.tif"])
geo_process(command="gdaldem", args=["slope", "dem.tif", "slope.tif"])
geo_process(command="gdal_translate", args=["-of", "GTiff", "input.vrt", "output.tif"])
```

### Step 4: Convert Deliverables
If the result needs a simpler output format, use:
```
geo_format_convert(input_path="result.tif", output_path="result.png")
```

## Conventions

- Inspect before processing.
- Reproject before area or distance analysis.
- Prefer GeoTIFF for raster outputs unless the user asks for a web-ready preview.
- Keep all generated outputs inside the workspace.
