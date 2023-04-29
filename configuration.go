package main

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"go.mau.fi/zeroconfig"
	"maunium.net/go/mautrix/id"
	"maunium.net/go/mautrix/util/dbutil"
)

type Configuration struct {
	// Authentication settings
	Homeserver   string    `yaml:"homeserver"`
	Username     id.UserID `yaml:"username"`
	PasswordFile string    `yaml:"password_file"`

	AutoJoin bool `yaml:"auto_join"`

	// Database settings
	Database dbutil.Config `yaml:"database"`

	// Logging configuration
	Logging zeroconfig.Config `yaml:"logging"`
}

func (c *Configuration) GetPassword(log *zerolog.Logger) (string, error) {
	log.Debug().Str("password_file", c.PasswordFile).Msg("reading password from file")
	buf, err := os.ReadFile(c.PasswordFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(buf)), nil
}
