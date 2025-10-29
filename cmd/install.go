package cmd

import (
	"fmt"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"npgo/internal/installer"
	"npgo/internal/lockfile"
	"npgo/internal/packagejson"
	"npgo/internal/registry"
	"npgo/internal/resolver"
	"npgo/internal/ui"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install [package]",
	Aliases: []string{"i"},
	Short:   "Install a package",
	Long: `Install downloads and links a package to node_modules.
If no package is specified, it installs dependencies from package.json.

Examples:
  npgo install express
  npgo install react@18.3.1
  npgo install             # Install from package.json`,
	Args: cobra.MinimumNArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		ui.Logo()

		if len(args) > 0 {
			installSinglePackage(args[0])
			return
		}

		installFromPackageJSON()
	},
}

var devFlag bool
var resolveConcurrency int

func init() {
	installCmd.Flags().BoolVarP(&devFlag, "dev", "D", false, "Install as dev dependency")
	installCmd.Flags().IntVarP(&resolveConcurrency, "concurrency", "c", 0, "resolver concurrency (0=auto)")
	rootCmd.AddCommand(installCmd)

}

func installSinglePackage(pkgSpec string) {
	ui.PrintHeader("Installing Package")

	name, version, err := parsePackageSpec(pkgSpec)
	if err != nil {
		ui.ErrorMessage(err)
		os.Exit(1)
	}

	startTime := time.Now()

	inst := installer.NewInstallerWithDebug("./node_modules", devFlag)

	ui.InstallStep("📦", fmt.Sprintf("Installing %s@%s...", name, version))
	spinner := ui.NewSpinner("Preparing installation")
	spinner.Start()
	time.Sleep(500 * time.Millisecond)
	spinner.Stop()

	resolvedVersion, err := inst.InstallPackage(name, version)
	if err != nil {
		ui.ErrorMessage(err)
		os.Exit(1)
	}

	if resolvedVersion != version {
		version = resolvedVersion
	}

	duration := time.Since(startTime)
	ui.SuccessMessage(name, version, duration.String())
}

func installFromPackageJSON() {
	ui.PrintHeader("Installing Dependencies")

	if _, err := os.Stat("package.json"); os.IsNotExist(err) {
		ui.Warning.Println("⚠️  No package.json found!")
		fmt.Println()
		ui.Info.Println("You can create one by running:")
		fmt.Println("  npgo init")
		fmt.Println()
		os.Exit(1)
	}

	pkg, err := packagejson.Read("package.json")
	if err != nil {
		ui.ErrorMessage(fmt.Errorf("failed to read package.json: %w", err))
		os.Exit(1)
	}

	if !pkg.HasDependencies() {
		ui.Info.Println("✅ No dependencies to install")
		fmt.Println()
		return
	}

	startTime := time.Now()

	ui.InstallStep("📋", fmt.Sprintf("Found %d dependencies to install", len(pkg.GetDependencies())))

	if resolveConcurrency == 0 {
		resolveConcurrency = autoConcurrency()
	}
	var resolvedCount int32
	res := resolver.NewResolverWithOptions(devFlag, resolveConcurrency, func(_ string) { atomic.AddInt32(&resolvedCount, 1) })
	spinner := ui.NewSpinner("Resolving dependencies...")
	spinner.Start()
	stopCh := make(chan struct{})

	if devFlag {
		ui.InstallStep("🛠️", "--dev enabled: verbose debug logs active")
		ui.InstallStep("🧩", fmt.Sprintf("Dependencies: %d, DevDependencies: %d", len(pkg.Dependencies), len(pkg.DevDependencies)))
	}
	rootSpecs := pkg.Dependencies
	if devFlag {
		rootSpecs = pkg.GetDependencies()
	}
	names := make([]string, 0, len(rootSpecs))
	for n := range rootSpecs {
		names = append(names, n)
	}
	go registry.PrefetchRegistry(names, resolveConcurrency)
	graph, err := res.BuildGraph(rootSpecs)
	if err != nil {
		spinner.Stop()
		close(stopCh)
		ui.ErrorMessage(err)
		os.Exit(1)
	}
	order, err := resolver.TopoOrder(graph)
	if err != nil {
		spinner.Stop()
		ui.ErrorMessage(err)
		os.Exit(1)
	}
	spinner.Stop()
	ui.InstallStep("✅", "Dependencies resolved (topo ordered)")
	if devFlag {
		ui.InstallStep("🔎", "Resolved packages:")
		for _, d := range order {
			ui.Muted.Printf("   - %s@%s (spec: %s)\n", d.Name, d.Resolved, d.Spec)
		}
	}

	inst := installer.NewInstallerWithDebug("./node_modules", devFlag)

	pkgs := make([]installer.PackageSpec, 0, len(order))
	for _, d := range order {
		pkgs = append(pkgs, installer.PackageSpec{Name: d.Name, Version: d.Resolved, TarballURL: d.TarballURL})
	}
	instSpinner := ui.NewSpinner("Installing packages (pipeline)...")
	instSpinner.Start()
	dw := resolveConcurrency
	if dw == 0 {
		dw = autoConcurrency()
	}
	lw := dw / 2
	if lw < 8 {
		lw = 8
	}
	if err := inst.InstallPipeline(pkgs, dw, lw); err != nil {
		instSpinner.Stop()
		ui.ErrorMessage(fmt.Errorf("pipeline install failed: %w", err))
		os.Exit(1)
	}
	instSpinner.Stop()
	ui.InstallStep("✅", "All packages installed")

	var lockPkgs []lockfile.PackageEntry
	for _, d := range order {
		lockPkgs = append(lockPkgs, lockfile.PackageEntry{
			Name: d.Name, Version: d.Resolved, Resolved: d.TarballURL, Integrity: "sha256", // TODO compute
		})
	}
	_ = lockfile.Save(".", &lockfile.LockFile{LockfileVersion: 1, Packages: lockPkgs})

	duration := time.Since(startTime)
	packageNames := make([]string, len(order))
	for i, dep := range order {
		packageNames[i] = dep.Name
	}
	ui.InstallSummary(packageNames, duration.String())
}

func autoConcurrency() int {
	cores := runtime.NumCPU()
	base := cores * 16
	if base < 64 {
		base = 64
	}
	if base > 256 {
		base = 256
	}
	return base
}
