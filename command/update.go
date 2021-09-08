package command

import (
	"fmt"
	"os"

	"github.com/blang/semver"
	updater "github.com/rhysd/go-github-selfupdate/selfupdate"
)

const APP_REPO = "svanas/nefertiti"

type (
	UpdateCommand struct {
		*CommandMeta
	}
)

func (c *UpdateCommand) development() bool {
	return c.AppVersion == "99.99.999"
}

func (c *UpdateCommand) Run(args []string) int {
	// do not mistakenly update the developer build
	if c.development() {
		fmt.Println("You are running the development build. Quitting.")
		return 0
	}

	// check for update
	latest, found, err := updater.DetectLatest(APP_REPO)
	if err != nil {
		return c.ReturnError(fmt.Errorf("error occurred while detecting version: %v", err))
	}

	v := semver.MustParse(c.AppVersion)
	if !found || latest.Version.LTE(v) {
		fmt.Println("You are running the latest version. Thank you for staying up-to-date!")
		return 0
	}

	// fetch the update and apply it
	exe, err := os.Executable()
	if err != nil {
		return c.ReturnError(fmt.Errorf("could not locate executable path"))
	}
	if err := updater.UpdateTo(latest.AssetURL, exe); err != nil {
		return c.ReturnError(fmt.Errorf("error occurred while updating binary: %v", err))
	}

	fmt.Printf("Updated to new version: %s\n", latest.Version)

	return 0
}

func (c *UpdateCommand) Help() string {
	return "Usage: ./nefertiti update"
}

func (c *UpdateCommand) Synopsis() string {
	return "Check for a new version."
}
