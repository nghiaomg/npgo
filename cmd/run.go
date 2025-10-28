package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"npgo/internal/packagejson"
	"npgo/internal/ui"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [script]",
	Short: "Run a script from package.json",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		script := args[0]
		runScript(script)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}

func runScript(script string) {
	ui.PrintHeader(fmt.Sprintf("Running script: %s", script))

	ui.InstallStep("🧠", "Reading package.json...")
	pkg, err := packagejson.Read("package.json")
	if err != nil {
		ui.ErrorMessage(fmt.Errorf("failed to read package.json: %w", err))
		os.Exit(1)
	}

	cmdStr, ok := pkg.Scripts[script]
	if !ok || strings.TrimSpace(cmdStr) == "" {
		ui.ErrorMessage(fmt.Errorf("Script '%s' not found in package.json", script))
		os.Exit(1)
	}

	ui.InstallStep("🚀", fmt.Sprintf("Running \"%s\" → %s", script, cmdStr))

	var execCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Prepend global node_modules to NODE_PATH for resolution
		// so node -r <module> can find global linked packages
		globalNM := os.ExpandEnv("%USERPROFILE%\\.npgo\\node_modules")
		env := os.Environ()
		hasNodePath := false
		for i, e := range env {
			if strings.HasPrefix(e, "NODE_PATH=") {
				env[i] = e + ";" + globalNM
				hasNodePath = true
				break
			}
		}
		if !hasNodePath {
			env = append(env, "NODE_PATH="+globalNM)
		}
		execCmd = exec.Command("cmd", "/C", cmdStr)
		execCmd.Env = env
	} else {
		// POSIX
		globalNM := os.ExpandEnv("$HOME/.npgo/node_modules")
		env := os.Environ()
		hasNodePath := false
		for i, e := range env {
			if strings.HasPrefix(e, "NODE_PATH=") {
				env[i] = e + ":" + globalNM
				hasNodePath = true
				break
			}
		}
		if !hasNodePath {
			env = append(env, "NODE_PATH="+globalNM)
		}
		execCmd = exec.Command("bash", "-c", cmdStr)
		execCmd.Env = env
	}
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin

	if err := execCmd.Run(); err != nil {
		ui.ErrorMessage(fmt.Errorf("Script \"%s\" failed: %v", script, err))
		os.Exit(1)
	}
}
