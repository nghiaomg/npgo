package cmd

import (
	"fmt"
	"os"
	"time"

	"npgo/internal/installer"
	"npgo/internal/lockfile"
	"npgo/internal/packagejson"
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

		// Check if installing a specific package
		if len(args) > 0 {
			installSinglePackage(args[0])
			return
		}

		// Install from package.json
		installFromPackageJSON()
	},
}

var devFlag bool

func init() {
	installCmd.Flags().BoolVarP(&devFlag, "dev", "D", false, "Install as dev dependency")
	rootCmd.AddCommand(installCmd)

	// NOTE: cobra does not support a global interceptor out-of-the-box.
	// Mapping of npgo <script> to npgo run <script> is handled by
	// encouraging usage of `npgo run <script>`. Future improvement: custom
	// PreRun hook at root to transform args.
}

// installSinglePackage installs a single package
func installSinglePackage(pkgSpec string) {
	ui.PrintHeader("Installing Package")

	// Parse package spec
	name, version, err := parsePackageSpec(pkgSpec)
	if err != nil {
		ui.ErrorMessage(err)
		os.Exit(1)
	}

	startTime := time.Now()

	// Create installer
	inst := installer.NewInstaller("./node_modules")

	// Show progress
	ui.InstallStep("📦", fmt.Sprintf("Installing %s@%s...", name, version))
	spinner := ui.NewSpinner("Preparing installation")
	spinner.Start()
	time.Sleep(500 * time.Millisecond)
	spinner.Stop()

	// Install package
	resolvedVersion, err := inst.InstallPackage(name, version)
	if err != nil {
		ui.ErrorMessage(err)
		os.Exit(1)
	}

	// Use resolved version for success message
	if resolvedVersion != version {
		version = resolvedVersion
	}

	// Show success
	duration := time.Since(startTime)
	ui.SuccessMessage(name, version, duration.String())
}

// installFromPackageJSON installs dependencies from package.json
func installFromPackageJSON() {
	ui.PrintHeader("Installing Dependencies")

	// Check if package.json exists
	if _, err := os.Stat("package.json"); os.IsNotExist(err) {
		ui.Warning.Println("⚠️  No package.json found!")
		fmt.Println()
		ui.Info.Println("You can create one by running:")
		fmt.Println("  npgo init")
		fmt.Println()
		os.Exit(1)
	}

	// Read package.json
	pkg, err := packagejson.Read("package.json")
	if err != nil {
		ui.ErrorMessage(fmt.Errorf("failed to read package.json: %w", err))
		os.Exit(1)
	}

	// Check if has dependencies
	if !pkg.HasDependencies() {
		ui.Info.Println("✅ No dependencies to install")
		fmt.Println()
		return
	}

	startTime := time.Now()

	// Show package info
	ui.InstallStep("📋", fmt.Sprintf("Found %d dependencies to install", len(pkg.GetDependencies())))

	// Create resolver
	res := resolver.NewResolver()
	spinner := ui.NewSpinner("Resolving dependencies...")
	spinner.Start()

	// Resolve dependencies
	if devFlag {
		ui.InstallStep("🛠️", "--dev enabled: verbose debug logs active")
		ui.InstallStep("🧩", fmt.Sprintf("Dependencies: %d, DevDependencies: %d", len(pkg.Dependencies), len(pkg.DevDependencies)))
	}

	deps, err := res.ResolveDependencies(pkg)
	if err != nil {
		spinner.Stop()
		ui.ErrorMessage(err)
		os.Exit(1)
	}

	// Resolve dev dependencies if flag is set
	if devFlag {
		devDeps, err := res.ResolveDevDependencies(pkg, true)
		if err != nil {
			spinner.Stop()
			ui.ErrorMessage(err)
			os.Exit(1)
		}
		deps = append(deps, devDeps...)
	}

	// Build dependency graph and compute topological order
	graph, err := res.BuildGraph(pkg.Dependencies)
	if err != nil {
		spinner.Stop()
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
	ui.InstallStep("✅", fmt.Sprintf("Resolved %d packages (topo ordered)", len(order)))
	if devFlag {
		ui.InstallStep("🔎", "Resolved packages:")
		for _, d := range order {
			ui.Muted.Printf("   - %s@%s (spec: %s)\n", d.Name, d.Resolved, d.Spec)
		}
	}

	// Create installer
	inst := installer.NewInstaller("./node_modules")

	// Install each package
	instSpinner := ui.NewSpinner("Installing packages...")
	instSpinner.Start()

	for i, dep := range order {
		instSpinner.Suffix = ui.Accent.Sprintf(" Installing %s...", dep.Name)
		if devFlag {
			ui.InstallStep("➡️", fmt.Sprintf("Installing %s@%s", dep.Name, dep.Resolved))
		}
		if _, err := inst.InstallPackage(dep.Name, dep.Resolved); err != nil {
			instSpinner.Stop()
			ui.ErrorMessage(fmt.Errorf("failed to install %s: %w", dep.Name, err))
			os.Exit(1)
		}
		// Show progress
		if (i+1)%5 == 0 {
			instSpinner.Suffix = ui.Accent.Sprintf(" Installed %d/%d packages", i+1, len(order))
		}
	}

	instSpinner.Stop()
	ui.InstallStep("✅", "All packages installed")

	// Write lockfile
	var pkgs []lockfile.PackageEntry
	for _, d := range order {
		pkgs = append(pkgs, lockfile.PackageEntry{
			Name: d.Name, Version: d.Resolved, Resolved: d.TarballURL, Integrity: "sha256", // TODO compute
		})
	}
	_ = lockfile.Save(".", &lockfile.LockFile{LockfileVersion: 1, Packages: pkgs})

	// Show summary
	duration := time.Since(startTime)
	packageNames := make([]string, len(order))
	for i, dep := range order {
		packageNames[i] = dep.Name
	}
	ui.InstallSummary(packageNames, duration.String())
}
