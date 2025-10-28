package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

// Colors for consistent theming
var (
	Primary = color.New(color.FgCyan, color.Bold)
	Success = color.New(color.FgGreen, color.Bold)
	Warning = color.New(color.FgYellow, color.Bold)
	Error   = color.New(color.FgRed, color.Bold)
	Info    = color.New(color.FgBlue, color.Bold)
	Muted   = color.New(color.FgHiBlack)
	Accent  = color.New(color.FgMagenta, color.Bold)
)

// Logo displays the NPGO logo with gradient effect
func Logo() {
	fmt.Println()
	Primary.Println("╔══════════════════════════════════════════════════════════════╗")
	Primary.Println("║                                                              ║")
	Primary.Println("║    ██╗  ██╗██████╗  ██████╗  ██████╗                        ║")
	Primary.Println("║    ██║  ██║██╔══██╗██╔════╝ ██╔═══██╗                       ║")
	Primary.Println("║    ██████╔╝██████╔╝██║  ███╗██║   ██║                       ║")
	Primary.Println("║    ██╔══██╗██╔═══╝ ██║   ██║██║   ██║                       ║")
	Primary.Println("║    ██║  ██║██║     ╚██████╔╝╚██████╔╝                       ║")
	Primary.Println("║    ╚═╝  ╚═╝╚═╝      ╚═════╝  ╚═════╝                        ║")
	Primary.Println("║                                                              ║")
	Primary.Println("║              🚀 Fastest Package Manager Ever!                ║")
	Primary.Println("║                                                              ║")
	Primary.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// Welcome displays welcome message
func Welcome() {
	Primary.Println("🌟 Welcome to NPGO - The Future of Package Management!")
	fmt.Println()
	Muted.Println("Built with ❤️  in Go for maximum speed and beauty")
	fmt.Println()
}

// Spinner creates a new spinner with custom styling
func NewSpinner(text string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Prefix = Accent.Sprint("⚡ ")
	s.Suffix = Primary.Sprint(" " + text)
	s.Color("cyan")
	return s
}

// ProgressBar creates a styled progress bar
func NewProgressBar(max int, description string) *progressbar.ProgressBar {
	bar := progressbar.NewOptions(max,
		progressbar.OptionSetDescription(Accent.Sprint("📦 ")+description),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerHead:    "█",
			SaucerPadding: "░",
			BarStart:      "│",
			BarEnd:        "│",
		}),
		progressbar.OptionOnCompletion(func() {
			fmt.Print("\n")
		}),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
	)
	return bar
}

// PackageInfo displays package information in a styled table
func PackageInfo(pkgName, version, size string) {
	fmt.Println()
	Info.Println("📦 Package Information:")

	// Create a simple info display
	fmt.Printf("   %s %s@%s\n", Primary.Sprint("Name:"), pkgName, version)
	fmt.Printf("   %s %s\n", Primary.Sprint("Size:"), size)
	fmt.Printf("   %s %s\n", Primary.Sprint("Status:"), Success.Sprint("Ready to install"))
	fmt.Println()
}

// InstallStep displays installation step with emoji and color
func InstallStep(step, description string) {
	fmt.Printf("%s %s\n", Accent.Sprint(step), description)
}

// SuccessMessage displays success message with celebration
func SuccessMessage(pkgName, version, duration string) {
	fmt.Println()

	Success.Println("╔══════════════════════════════════════════════════════════════╗")
	Success.Println("║                                                              ║")
	Success.Printf("║                    🎉 SUCCESS! 🎉                              ║\n")
	Success.Println("║                                                              ║")
	Success.Printf("║    ✅ %s@%s installed successfully!                    ║\n", pkgName, version)
	Success.Printf("║    ⚡ Completed in %s                                    ║\n", duration)
	Success.Println("║    🚀 Ready to use!                                        ║")
	Success.Println("║                                                              ║")
	Success.Printf("║    %s                                                      ║\n", Success.Sprint("Happy coding! 🎯"))
	Success.Println("║                                                              ║")
	Success.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// ErrorMessage displays error message with styling
func ErrorMessage(err error) {
	fmt.Println()
	Error.Printf("❌ Error: %v\n", err)
	fmt.Println()
}

// CacheInfo displays cache information
func CacheInfo(cachePath, extractPath string) {
	fmt.Println()
	Info.Println("💾 Cache Information:")
	fmt.Printf("   %s %s\n", Muted.Sprint("Tarball:"), cachePath)
	fmt.Printf("   %s %s\n", Muted.Sprint("Extracted:"), extractPath)
	fmt.Println()
}

// InstallSummary displays installation summary
func InstallSummary(packages []string, totalTime string) {
	fmt.Println()

	Info.Println("╔══════════════════════════════════════════════════════════════╗")
	Info.Println("║                                                              ║")
	Info.Println("║                📊 INSTALLATION SUMMARY                      ║")
	Info.Println("║                                                              ║")
	Info.Printf("║    📦 Packages installed: %d                                 ║\n", len(packages))
	Info.Printf("║    ⏱️  Total time: %s                                        ║\n", totalTime)
	Info.Println("║    💾 Cache location: ~/.npgo/                              ║")
	Info.Println("║                                                              ║")
	Info.Printf("║    %s                                                      ║\n", Success.Sprint("All packages ready! 🚀"))
	Info.Println("║                                                              ║")
	Info.Println("╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

// LoadingAnimation displays a loading animation
func LoadingAnimation(text string, duration time.Duration) {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

	for i := 0; i < int(duration.Seconds()*10); i++ {
		frame := frames[i%len(frames)]
		fmt.Printf("\r%s %s", Accent.Sprint(frame), text)
		time.Sleep(100 * time.Millisecond)
	}
	fmt.Print("\r" + strings.Repeat(" ", len(text)+3) + "\r")
}

// ClearScreen clears the terminal screen
func ClearScreen() {
	fmt.Print("\033[2J\033[H")
}

// PrintSeparator prints a styled separator
func PrintSeparator() {
	fmt.Println()
	Muted.Println(strings.Repeat("─", 60))
	fmt.Println()
}

// PrintHeader prints a styled header
func PrintHeader(title string) {
	fmt.Println()
	Primary.Println(strings.Repeat("═", len(title)+4))
	Primary.Printf("  %s  \n", title)
	Primary.Println(strings.Repeat("═", len(title)+4))
	fmt.Println()
}

// CheckMark returns a styled checkmark
func CheckMark() string {
	return Success.Sprint("✅")
}

// CrossMark returns a styled cross mark
func CrossMark() string {
	return Error.Sprint("❌")
}

// Arrow returns a styled arrow
func Arrow() string {
	return Accent.Sprint("→")
}

// Bullet returns a styled bullet point
func Bullet() string {
	return Primary.Sprint("•")
}
