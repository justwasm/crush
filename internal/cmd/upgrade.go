package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/charmbracelet/crush/internal/version"
	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade crush to the latest version",
	Long:  `Detect the latest crush release on GitHub and replace the running binary in place.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		currentVersion := version.Version
		log.Printf("current version: %s", currentVersion)
		return upgrade(currentVersion)
	},
}

func upgrade(currentVersion string) error {
	latest, found, err := selfupdate.DetectLatest(context.TODO(), selfupdate.ParseSlug("justwasm/crush"))
	if err != nil {
		return fmt.Errorf("error occurred while detecting version: %v", err)
	}
	if !found {
		return fmt.Errorf("latest version for %s/%s could not be found from github repository", runtime.GOOS, runtime.GOARCH)
	}

	log.Printf("found latest: %s", latest.AssetURL)

	if currentVersion != "" && latest.LessOrEqual(currentVersion) {
		log.Printf("Current version (%s) is the latest", currentVersion)
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return errors.New("could not locate executable path")
	}
	if err := selfupdate.UpdateTo(context.TODO(), latest.AssetURL, latest.AssetName, exe); err != nil {
		return fmt.Errorf("error occurred while updating binary: %v", err)
	}
	log.Printf("Successfully updated to version %s", latest.Version())
	return nil
}
