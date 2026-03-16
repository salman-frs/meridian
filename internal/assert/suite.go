package assert

import (
	"fmt"
	"os"

	"github.com/salman-frs/meridian/internal/generator"
	"github.com/salman-frs/meridian/internal/model"
	"gopkg.in/yaml.v3"
)

func LoadSuite(path string, runID string) (model.AssertionFile, error) {
	if path == "" {
		return model.AssertionFile{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return model.AssertionFile{}, err
	}
	var file model.AssertionFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return model.AssertionFile{}, err
	}
	for _, fixture := range file.Fixtures {
		if !generator.IsKnownFixture(fixture) {
			return model.AssertionFile{}, fmt.Errorf("unknown fixture %q", fixture)
		}
	}
	assertions := make([]model.AssertionSpec, 0, len(file.Assertions))
	for _, spec := range file.Assertions {
		assertions = append(assertions, applyDefaults(spec, file.Defaults, runID))
	}
	file.Assertions = assertions
	contracts := make([]model.ContractSpec, 0, len(file.Contracts))
	for _, spec := range file.Contracts {
		contracts = append(contracts, applyContractDefaults(spec, file.Defaults, file.Fixtures, runID))
	}
	file.Contracts = contracts
	return file, nil
}

func LoadCustomAssertions(path string, runID string) ([]model.AssertionSpec, error) {
	suite, err := LoadSuite(path, runID)
	if err != nil {
		return nil, err
	}
	return suite.Assertions, nil
}
