package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/uoregon-libraries/gopkg/bashconf"
)

// Config is used to define the configuration for both the indexer and the web server
type Config struct {
	BindAddress           string `setting:"BIND_ADDRESS"`
	WebPath               string `setting:"WEBPATH" type:"url"`
	Approot               string `setting:"APPROOT" type:"path"`
	DARoot                string `setting:"DARK_ARCHIVE_PATH" type:"path"`
	PathFormat            []PathToken
	PathFormatString      string `setting:"ARCHIVE_PATH_FORMAT"`
	InventoryPattern      string `setting:"INVENTORY_FILE_GLOB"`
	ArchiveOutputLocation string `setting:"ARCHIVE_OUTPUT_LOCATION" type:"path"`
	ArchiveLifetimeDays   int    `setting:"ARCHIVE_LIFETIME_DAYS" type:"int"`
}

// Read opens the given file and reads its configuration
func Read(filename string) (*Config, error) {
	var conf = bashconf.New()
	conf.EnvironmentPrefix("HL_")

	var err = conf.ParseFile(filename)
	if err != nil {
		return nil, err
	}
	var c = &Config{}
	err = conf.Store(c)
	if err != nil {
		return nil, err
	}
	err = c.parsePathFormat()
	if err != nil {
		return nil, fmt.Errorf("invalid ARCHIVE_PATH_FORMAT %q: %s", c.PathFormatString, err)
	}

	return c, nil
}

func (c *Config) parsePathFormat() error {
	var formatParts = strings.Split(c.PathFormatString, string(os.PathSeparator))
	var hasProject bool
	var hasDate bool
	for _, part := range formatParts {
		switch part {
		case "ignore":
			c.PathFormat = append(c.PathFormat, Ignored)

		case "project":
			if hasProject {
				return fmt.Errorf(`"project" must be specified exactly once`)
			}
			hasProject = true
			c.PathFormat = append(c.PathFormat, Project)

		case "date":
			if hasDate {
				return fmt.Errorf(`"date" must be specified exactly once`)
			}
			hasDate = true
			c.PathFormat = append(c.PathFormat, Date)

		default:
			return fmt.Errorf("unknown keyword %q", part)
		}
	}
	if !hasProject {
		return fmt.Errorf(`"project" must be specified exactly once`)
	}
	if !hasDate {
		return fmt.Errorf(`"date" must be specified exactly once`)
	}

	return nil
}
