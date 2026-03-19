package tools

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// gdalAllowedCommands is the whitelist of GDAL commands that geo_process may execute.
var gdalAllowedCommands = map[string]bool{
	"gdal_translate": true,
	"gdalwarp":       true,
	"ogr2ogr":        true,
	"gdal_rasterize": true,
	"gdal_contour":   true,
	"gdalbuildvrt":   true,
	"gdaldem":        true,
	"gdal_grid":      true,
	"gdal_create":    true,
	"nearblack":      true,
	"gdalsrsinfo":    true,
	"gdalinfo":       true,
	"ogrinfo":        true,
	"gdalmdiminfo":   true,
}

// shellMetaChars contains characters that could be used for shell injection.
var shellMetaChars = []string{";", "|", "&&", "||", "`", "$(", "\n", "\r"}

// resolveGdalCommand returns the full path to a GDAL executable.
// If binDir is non-empty, it prepends the directory; otherwise it relies on PATH.
func resolveGdalCommand(binDir, command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(command), ".exe") {
		command += ".exe"
	}
	binDir = strings.TrimSpace(binDir)
	if binDir != "" {
		return filepath.Join(binDir, command)
	}
	return command
}

// isGdalAvailable checks if a GDAL command is available via PATH or the configured binDir.
func isGdalAvailable(binDir, command string) bool {
	resolved := resolveGdalCommand(binDir, command)
	_, err := exec.LookPath(resolved)
	return err == nil
}

// runGdalCommand executes a GDAL command with the given arguments and returns stdout/stderr.
func runGdalCommand(ctx context.Context, binDir string, command string, args []string, workDir string, timeoutSec int) (stdout, stderr string, exitCode int, err error) {
	resolved := resolveGdalCommand(binDir, command)

	timeout := time.Duration(timeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, resolved, args...)
	if workDir != "" {
		cmd.Dir = workDir
	}

	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()
	exitCode = 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", "", 1, runErr
		}
	}

	return stdoutBuf.String(), stderrBuf.String(), exitCode, nil
}

// validateGeoFilePath checks that the given file path is within the workspace directory.
func validateGeoFilePath(path, workspaceDir string, restrictToWorkspace bool) error {
	if !restrictToWorkspace || workspaceDir == "" {
		return nil
	}
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("path is required")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	absWorkspace, err := filepath.Abs(workspaceDir)
	if err != nil {
		return fmt.Errorf("invalid workspace path: %w", err)
	}

	rel, err := filepath.Rel(absWorkspace, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("path %q is outside workspace %q", path, workspaceDir)
	}
	return nil
}

// geoFormatInfo describes a geospatial file format.
type geoFormatInfo struct {
	Name     string // GDAL driver name, e.g. "GTiff"
	IsRaster bool
	IsVector bool
}

// knownGeoFormats maps lowercase file extensions to format info.
var knownGeoFormats = map[string]geoFormatInfo{
	".tif":     {Name: "GTiff", IsRaster: true},
	".tiff":    {Name: "GTiff", IsRaster: true},
	".geotiff": {Name: "GTiff", IsRaster: true},
	".img":     {Name: "HFA", IsRaster: true},
	".hdf":     {Name: "HDF4", IsRaster: true},
	".hdf5":    {Name: "HDF5", IsRaster: true},
	".nc":      {Name: "netCDF", IsRaster: true},
	".vrt":     {Name: "VRT", IsRaster: true},
	".asc":     {Name: "AAIGrid", IsRaster: true},
	".png":     {Name: "PNG", IsRaster: true},
	".jpg":     {Name: "JPEG", IsRaster: true},
	".jpeg":    {Name: "JPEG", IsRaster: true},
	".jp2":     {Name: "JP2OpenJPEG", IsRaster: true},
	".shp":     {Name: "ESRI Shapefile", IsVector: true},
	".geojson": {Name: "GeoJSON", IsVector: true},
	".json":    {Name: "GeoJSON", IsVector: true},
	".gpkg":    {Name: "GPKG", IsRaster: true, IsVector: true},
	".gdb":     {Name: "OpenFileGDB", IsVector: true},
	".kml":     {Name: "KML", IsVector: true},
	".kmz":     {Name: "LIBKML", IsVector: true},
	".gml":     {Name: "GML", IsVector: true},
	".csv":     {Name: "CSV", IsVector: true},
	".gpx":     {Name: "GPX", IsVector: true},
	".tab":     {Name: "MapInfo File", IsVector: true},
	".mif":     {Name: "MapInfo File", IsVector: true},
	".dxf":     {Name: "DXF", IsVector: true},
	".fgb":     {Name: "FlatGeobuf", IsVector: true},
	".parquet": {Name: "Parquet", IsVector: true},
}

// detectGeoFormat detects the geospatial format from a file extension.
func detectGeoFormat(path string) (geoFormatInfo, bool) {
	ext := strings.ToLower(filepath.Ext(path))
	info, ok := knownGeoFormats[ext]
	return info, ok
}

// containsShellMeta checks if any argument contains shell metacharacters.
func containsShellMeta(args []string) (bool, string) {
	for _, arg := range args {
		for _, meta := range shellMetaChars {
			if strings.Contains(arg, meta) {
				return true, fmt.Sprintf("argument %q contains shell metacharacter %q", arg, meta)
			}
		}
	}
	return false, ""
}

// gdalNotFoundError returns a user-friendly error when GDAL is not installed.
func gdalNotFoundError(command string) string {
	return fmt.Sprintf("GDAL command %q not found. "+
		"Please install GDAL:\n"+
		"  - Windows: OSGeo4W (https://trac.osgeo.org/osgeo4w/) or conda install -c conda-forge gdal\n"+
		"  - Linux:   apt install gdal-bin / yum install gdal\n"+
		"  - macOS:   brew install gdal\n"+
		"If GDAL is installed in a custom location, set tools.geo.gdal_bin_dir in config.",
		command)
}
