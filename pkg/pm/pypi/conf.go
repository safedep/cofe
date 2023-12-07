package pypi

import (
	"path"

	"github.com/bigkevmcd/go-configparser"
	"github.com/mitchellh/go-homedir"
	"github.com/safedep/dry/log"
)

type IndexUrlsConf struct {
	ReadStdPipConf         bool
	DisableDefaultIndexUrl bool
}

// ParsePipConfAndExtractIndexURLs reads pip.conf and extracts index URLs.
func parsePipConfAndExtractIndexURLs(filename string) ([]string, error) {
	// Create a new ConfigParser instance
	p, err := configparser.NewConfigParserFromFile(filename)
	if err != nil {
		return nil, err
	}

	// Get the list of sections in the configuration
	sections := p.Sections()

	var indexURLs []string

	// Iterate over sections and extract index URLs
	for _, section := range sections {
		optionValue, err := p.Get(section, "index-url")
		if err == nil {
			indexURLs = append(indexURLs, optionValue)
		}

		optionValue, err = p.Get(section, "extra-index-url")
		if err == nil {
			indexURLs = append(indexURLs, optionValue)
		}

	}

	return indexURLs, nil
}

func GetIndexURLsFromDefaultPipConf() ([]string, error) {
	var indexUrls []string
	home, err := homedir.Dir()
	if err != nil {
		return indexUrls, nil
	}
	pipconf := path.Join(home, ".pip/pip.conf")
	indexUrls, err = parsePipConfAndExtractIndexURLs(pipconf)
	if err != nil {
		log.Debugf("Error while reading default pip conf file %s", err)
		return indexUrls, err
	}

	return indexUrls, nil
}

func GetIndexURLs(iconf IndexUrlsConf) ([]string, error) {
	var indexUrls []string
	var err error
	if iconf.ReadStdPipConf {
		indexUrls, err = GetIndexURLsFromDefaultPipConf()
		if err != nil {
			return indexUrls, err
		}
	}

	if !iconf.DisableDefaultIndexUrl {
		indexUrls = append(indexUrls, "https://pypi.org/pypi")
	}

	return indexUrls, nil
}
