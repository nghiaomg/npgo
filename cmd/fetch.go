package cmd

import (
	"fmt"
	"os"
	"time"

	"npgo/internal/cache"
	"npgo/internal/extractor"
	"npgo/internal/registry"
	"npgo/internal/ui"

	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch [package]@[version]",
	Short: "Fetch a package from npm registry",
	Long: `Fetch downloads a package from npm registry and caches it locally.
This command downloads the package tarball and extracts it to the cache directory.

Examples:
  npgo fetch express@4.18.2
  npgo fetch lodash@latest
  npgo fetch react`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pkgSpec := args[0]

		// Show logo and start
		ui.Logo()
		ui.PrintHeader("Fetching Package")

		startTime := time.Now()

		// Parse package name and version
		pkgName, version, err := parsePackageSpec(pkgSpec)
		if err != nil {
			ui.ErrorMessage(err)
			os.Exit(1)
		}

		ui.InstallStep("🔍", fmt.Sprintf("Parsing package specification: %s@%s", pkgName, version))

		// Check if already cached
		cachePath := cache.GetCachePath(pkgName, version)
		if cache.Exists(cachePath) {
			ui.InstallStep("✅", fmt.Sprintf("Package already cached: %s@%s", pkgName, version))
			ui.CacheInfo(cachePath, cache.GetExtractPath(pkgName, version))
			return
		}

		// Create spinner for metadata fetching
		s := ui.NewSpinner("Fetching metadata from npm registry...")
		s.Start()

		// Fetch metadata from npm registry
		metadata, err := registry.FetchMetadata(pkgName, version)
		if err != nil {
			s.Stop()
			ui.ErrorMessage(fmt.Errorf("failed to fetch metadata: %w", err))
			os.Exit(1)
		}

		s.Stop()
		ui.InstallStep("✅", "Metadata fetched successfully")

		// Show package info
		ui.PackageInfo(metadata.Name, metadata.Version, "Calculating...")

		// Create progress bar for download
		progressBar := ui.NewProgressBar(100, fmt.Sprintf("Downloading %s@%s", pkgName, metadata.Version))

		// Download tarball
		tarballPath, err := registry.DownloadTarball(metadata.TarballURL, pkgName, metadata.Version)
		if err != nil {
			progressBar.Close()
			ui.ErrorMessage(fmt.Errorf("failed to download tarball: %w", err))
			os.Exit(1)
		}

		// Simulate progress
		for i := 0; i <= 100; i += 10 {
			progressBar.Set(i)
			time.Sleep(50 * time.Millisecond)
		}
		progressBar.Close()

		ui.InstallStep("✅", "Tarball downloaded successfully")

		// Extract package
		extractSpinner := ui.NewSpinner("Extracting package...")
		extractSpinner.Start()

		extractPath := cache.GetExtractPath(pkgName, metadata.Version)
		if err := extractor.ExtractTarGz(tarballPath, extractPath); err != nil {
			extractSpinner.Stop()
			ui.ErrorMessage(fmt.Errorf("failed to extract package: %w", err))
			os.Exit(1)
		}

		extractSpinner.Stop()
		ui.InstallStep("✅", "Package extracted successfully")

		// Calculate duration
		duration := time.Since(startTime)

		// Show success message
		ui.SuccessMessage(metadata.Name, metadata.Version, duration.String())
		ui.CacheInfo(cachePath, extractPath)
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)
}

func parsePackageSpec(spec string) (name, version string, err error) {
	// Simple parser for package@version format
	for i, char := range spec {
		if char == '@' {
			name = spec[:i]
			version = spec[i+1:]
			return name, version, nil
		}
	}
	return spec, "latest", nil
}
