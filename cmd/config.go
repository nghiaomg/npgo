package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show npgo config paths",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		cacheDir := filepath.Join(home, ".npgo")
		fmt.Println("NPGO Config:")
		fmt.Println("  Home:", home)
		fmt.Println("  Cache Root:", cacheDir)
		fmt.Println("  Cache Tarballs:", filepath.Join(cacheDir, "cache"))
		fmt.Println("  Extracted:", filepath.Join(cacheDir, "extracted"))
		fmt.Println("  CAS Store:", filepath.Join(cacheDir, "store", "v3"))
		fmt.Println("  Global node_modules:", filepath.Join(cacheDir, "node_modules"))
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
