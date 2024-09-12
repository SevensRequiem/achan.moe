package update

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"

	"github.com/creativeprojects/go-selfupdate"
)

// detectLatestVersion detects the latest version from the GitHub repository.
func detectLatestVersion(ctx context.Context) (*selfupdate.Release, error) {
	fmt.Println("Detecting latest version")
	latest, found, err := selfupdate.DetectLatest(ctx, selfupdate.ParseSlug("SevensRequiem/achan.moe"))
	if err != nil {
		return nil, fmt.Errorf("error occurred while detecting version: %w", err)
	}
	if !found {
		return nil, fmt.Errorf("latest version for %s/%s could not be found from GitHub repository", runtime.GOOS, runtime.GOARCH)
	}
	return latest, nil
}

// Update updates the current executable to the latest version.
func Update(ctx context.Context, version string) error {
	fmt.Println("Checking for updates")
	latest, err := detectLatestVersion(ctx)
	if err != nil {
		return err
	}

	if latest.LessOrEqual(version) {
		log.Printf("Current version (%s) is the latest", version)
		return nil
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return errors.New("could not locate executable path")
	}
	if err := selfupdate.UpdateTo(ctx, latest.AssetURL, latest.AssetName, exe); err != nil {
		return fmt.Errorf("error occurred while updating binary: %w", err)
	}
	log.Printf("Successfully updated to version %s", latest.Version())
	return nil
}

// CheckForUpdate checks if there is a new version available.
func CheckForUpdate(ctx context.Context, version string) error {
	fmt.Println("Checking for updates")
	latest, err := detectLatestVersion(ctx)
	if err != nil {
		return err
	}

	if latest.LessOrEqual(version) {
		log.Printf("Current version (%s) is the latest", version)
		return nil
	}

	log.Printf("New version %s is available", latest.Version())
	return nil
}
