package cliconfig

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/buildkite/agent/utils"
)

type iniSource struct {
	paths []string
}

func NewIniSource(paths ...string) Source {
	return &iniSource{paths}
}

func (s *iniSource) Values() ([]Value, error) {
	for _, path := range s.paths {
		if _, err := os.Stat(path); os.IsExist(err) {
			// Normalize and save the path
			absolutePath, err := utils.NormalizeFilePath(path)
			if err != nil {
				return nil, err
			}

			// Turn it into a source
			inf := &iniFileSource{
				path: absolutePath,
			}

			return inf.Values()
		}
	}

	return nil, nil
}

type iniFileSource struct {
	path string
}

func (fs iniFileSource) Type() string {
	return "inifile"
}

func (fs iniFileSource) Path() string {
	return fs.path
}

func (fs *iniFileSource) Values() ([]Value, error) {
	var values []Value

	f, err := os.Open(fs.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// The timestamp to store in the values
	timestamp := time.Now()

	// Whether to ignore the line
	isIgnoredLine := func(line string) bool {
		trimmedLine := strings.Trim(line, " \n\t")
		return len(trimmedLine) == 0 || strings.HasPrefix(trimmedLine, "#")
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		if isIgnoredLine(line) {
			continue
		}

		key, value, err := parseIniLine(line)
		if err != nil {
			return nil, err
		}

		values = append(values, Value{
			Name:      key,
			Contents:  value,
			Source:    fs,
			Timestamp: timestamp,
		})
	}

	return values, nil
}

// Derived from https://github.com/joho/godotenv/blob/master/godotenv.go
func parseIniLine(line string) (key string, value string, err error) {
	if len(line) == 0 {
		err = errors.New("zero length string")
		return
	}

	// ditch the comments (but keep quoted hashes)
	if strings.Contains(line, "#") {
		segmentsBetweenHashes := strings.Split(line, "#")
		quotesAreOpen := false
		segmentsToKeep := make([]string, 0)
		for _, segment := range segmentsBetweenHashes {
			if strings.Count(segment, "\"") == 1 || strings.Count(segment, "'") == 1 {
				if quotesAreOpen {
					quotesAreOpen = false
					segmentsToKeep = append(segmentsToKeep, segment)
				} else {
					quotesAreOpen = true
				}
			}

			if len(segmentsToKeep) == 0 || quotesAreOpen {
				segmentsToKeep = append(segmentsToKeep, segment)
			}
		}

		line = strings.Join(segmentsToKeep, "#")
	}

	// now split key from value
	splitString := strings.SplitN(line, "=", 2)

	if len(splitString) != 2 {
		// try yaml mode!
		splitString = strings.SplitN(line, ":", 2)
	}

	if len(splitString) != 2 {
		err = errors.New("Can't separate key from value")
		return
	}

	// Parse the key
	key = strings.Trim(splitString[0], " ")

	// Parse the value
	value = strings.Trim(splitString[1], " ")

	// check if we've got quoted values
	if strings.Count(value, "\"") == 2 || strings.Count(value, "'") == 2 {
		// pull the quotes off the edges
		value = strings.Trim(value, "\"'")

		// expand quotes
		value = strings.Replace(value, "\\\"", "\"", -1)
		// expand newlines
		value = strings.Replace(value, "\\n", "\n", -1)
	}

	return
}
