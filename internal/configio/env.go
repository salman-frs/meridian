package configio

import (
	"bufio"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
)

var envPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

type EnvSet struct {
	Values     map[string]string
	References []model.EnvReference
	Missing    []string
}

func LoadEnv(envFile string, inline []string, includeOS bool) (map[string]string, error) {
	values := map[string]string{}
	if includeOS {
		for _, entry := range os.Environ() {
			key, value, ok := strings.Cut(entry, "=")
			if ok {
				values[key] = value
			}
		}
	}
	if envFile != "" {
		file, err := os.Open(envFile)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if ok {
				values[strings.TrimSpace(key)] = strings.TrimSpace(value)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	for _, entry := range inline {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			values[strings.TrimSpace(key)] = value
		}
	}
	return values, nil
}

func InterpolateValue(input string, values map[string]string) (string, []model.EnvReference, []string) {
	refs := []model.EnvReference{}
	missingSet := map[string]struct{}{}
	output := envPattern.ReplaceAllStringFunc(input, func(match string) string {
		name := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		name = strings.TrimPrefix(name, "env:")
		name, _, _ = strings.Cut(name, ":-")
		name = strings.TrimSpace(name)
		value, ok := values[name]
		refs = append(refs, model.EnvReference{
			Name:     name,
			Original: match,
			HasValue: ok,
		})
		if !ok {
			missingSet[name] = struct{}{}
			return match
		}
		return value
	})
	missing := make([]string, 0, len(missingSet))
	for name := range missingSet {
		missing = append(missing, name)
	}
	sort.Strings(missing)
	return output, refs, missing
}
