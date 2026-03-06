package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
)

var Version = "dev"

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade krawl to the latest version",
	RunE:  runUpgrade,
}

func init() {
	rootCmd.AddCommand(upgradeCmd)
	rootCmd.Version = Version
}

type ghRelease struct {
	TagName string `json:"tag_name"`
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	resp, err := http.Get("https://api.github.com/repos/devforward/krawl/releases/latest")
	if err != nil {
		return fmt.Errorf("checking latest version: %w", err)
	}
	defer resp.Body.Close()

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("parsing release info: %w", err)
	}

	if release.TagName == Version {
		fmt.Printf("Already on the latest version (%s)\n", Version)
		return nil
	}

	fmt.Printf("Current: %s → Latest: %s\n", Version, release.TagName)
	fmt.Println("Upgrading...")

	name := fmt.Sprintf("krawl-%s-%s", runtime.GOOS, runtime.GOARCH)
	url := fmt.Sprintf("https://github.com/devforward/krawl/releases/latest/download/%s", name)

	bin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}

	// Download to temp directory first to avoid permission issues
	tmpFile, err := os.CreateTemp("", "krawl-upgrade-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmp := tmpFile.Name()
	tmpFile.Close()

	out, err := exec.Command("curl", "-fSL", "-o", tmp, url).CombinedOutput()
	if err != nil {
		return fmt.Errorf("downloading: %s\n%s", err, out)
	}

	if err := os.Chmod(tmp, 0755); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("chmod: %w", err)
	}

	if err := os.Rename(tmp, bin); err != nil {
		// Try with sudo if permission denied
		out, sudoErr := exec.Command("sudo", "mv", tmp, bin).CombinedOutput()
		if sudoErr != nil {
			os.Remove(tmp)
			return fmt.Errorf("replacing binary: %s\n%s", sudoErr, out)
		}
	}

	fmt.Printf("Upgraded to %s\n", release.TagName)
	return nil
}
