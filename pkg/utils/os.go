package utils

import (
	"os"
	"os/user"
)

// GetOSUsername get current host id, hostname + username.
func GetOSUsername() string {
	username := "unknown"
	curUser, err := user.Current()
	if err == nil {
		username = curUser.Username
	}
	return username
}

// GetUltrafoxHome get UltraFox default home directory, is $HOME/.ultrafox
func GetUltrafoxHome() string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return dirname + "/.ultrafox"
}
