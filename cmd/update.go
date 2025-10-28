package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"npgo/internal/ui"
	"npgo/internal/updater"

	"github.com/spf13/cobra"
)

var currentVersion = "v0.0.1"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update npgo to the latest release",
	Run: func(cmd *cobra.Command, args []string) {
		ui.PrintHeader("Update NPGO")
		latest, hasNew, err := updater.CheckUpdate(currentVersion)
		if err != nil {
			ui.ErrorMessage(fmt.Errorf("failed to check update: %w", err))
			return
		}
		if !hasNew {
			ui.InstallStep("✅", fmt.Sprintf("npgo is up to date: %s", currentVersion))
			return
		}
		ui.InstallStep("🚀", fmt.Sprintf("New version %s available (current: %s)", latest, currentVersion))
		ui.InstallStep("⬇️", "Downloading latest binary...")

		tmpDir := os.TempDir()
		binPath, tag, err := updater.DownloadLatest(tmpDir)
		if err != nil {
			ui.ErrorMessage(err)
			return
		}

		// Replace current binary (best-effort). On Windows, replacing running exe is not allowed.
		exe, _ := os.Executable()
		target := exe
		if runtime.GOOS == "windows" {
			// Write alongside as npgo.new.exe and instruct user
			target = filepath.Join(filepath.Dir(exe), "npgo.new.exe")
		}
		if err := copyFile(binPath, target); err != nil {
			ui.ErrorMessage(fmt.Errorf("failed to place binary: %w", err))
			return
		}
		if runtime.GOOS == "windows" {
			ui.InstallStep("ℹ️", "On Windows, please close this terminal and rename npgo.new.exe to npgo.exe manually (admin may be required).")
		}
		ui.SuccessMessage("npgo", tag, "")
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}

// local copy helper (avoid circular import)
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
