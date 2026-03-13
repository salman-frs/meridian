package configio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/salman-frs/meridian/internal/model"
	"gopkg.in/yaml.v3"
)

type LoadOptions struct {
	ConfigPath string
	ConfigDir  string
	EnvFile    string
	EnvInline  []string
}

func LoadConfig(opts LoadOptions) (model.ConfigModel, error) {
	if opts.ConfigPath == "" && opts.ConfigDir == "" {
		return model.ConfigModel{}, errors.New("either --config or --config-dir is required")
	}

	env, err := LoadEnv(opts.EnvFile, opts.EnvInline, true)
	if err != nil {
		return model.ConfigModel{}, err
	}

	merged := map[string]any{}
	sourcePaths := []string{}
	if opts.ConfigPath != "" {
		content, err := os.ReadFile(opts.ConfigPath)
		if err != nil {
			return model.ConfigModel{}, err
		}
		doc, refs, missing, err := parseYAMLDocument(opts.ConfigPath, string(content), env)
		if err != nil {
			return model.ConfigModel{}, err
		}
		merged = mergeMaps(merged, doc)
		sourcePaths = append(sourcePaths, opts.ConfigPath)
		cfg := normalizeConfig(sourcePaths, merged)
		cfg.EnvReferences = append(cfg.EnvReferences, refs...)
		cfg.MissingEnvNames = append(cfg.MissingEnvNames, missing...)
		return dedupeEnv(cfg), nil
	}

	entries, err := os.ReadDir(opts.ConfigDir)
	if err != nil {
		return model.ConfigModel{}, err
	}
	paths := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			paths = append(paths, filepath.Join(opts.ConfigDir, name))
		}
	}
	sort.Strings(paths)
	allRefs := []model.EnvReference{}
	allMissing := []string{}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return model.ConfigModel{}, err
		}
		doc, refs, missing, err := parseYAMLDocument(path, string(content), env)
		if err != nil {
			return model.ConfigModel{}, err
		}
		merged = mergeMaps(merged, doc)
		sourcePaths = append(sourcePaths, path)
		allRefs = append(allRefs, refs...)
		allMissing = append(allMissing, missing...)
	}
	cfg := normalizeConfig(sourcePaths, merged)
	cfg.EnvReferences = allRefs
	cfg.MissingEnvNames = allMissing
	return dedupeEnv(cfg), nil
}

func parseYAMLDocument(path string, content string, env map[string]string) (map[string]any, []model.EnvReference, []string, error) {
	interpolated, refs, missing := interpolateDocument(content, env)
	for i := range refs {
		refs[i].SourcePath = path
		if refs[i].SourceKey == "" {
			refs[i].SourceKey = "$"
		}
	}
	raw := map[string]any{}
	if err := yaml.Unmarshal([]byte(interpolated), &raw); err != nil {
		return nil, nil, nil, err
	}
	return normalizeMap(raw), refs, missing, nil
}

func interpolateDocument(content string, env map[string]string) (string, []model.EnvReference, []string) {
	out, refs, missing := InterpolateValue(content, env)
	return out, refs, missing
}

func mergeMaps(dst map[string]any, src map[string]any) map[string]any {
	merged := map[string]any{}
	for key, value := range dst {
		merged[key] = value
	}
	for key, value := range src {
		if srcMap, ok := value.(map[string]any); ok {
			existing, _ := merged[key].(map[string]any)
			merged[key] = mergeMaps(existing, srcMap)
			continue
		}
		merged[key] = value
	}
	return merged
}

func normalizeMap(input map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			out[key] = normalizeMap(typed)
		case map[any]any:
			converted := map[string]any{}
			for mk, mv := range typed {
				converted[toString(mk)] = mv
			}
			out[key] = normalizeMap(converted)
		case []any:
			items := make([]any, 0, len(typed))
			for _, item := range typed {
				if itemMap, ok := item.(map[string]any); ok {
					items = append(items, normalizeMap(itemMap))
					continue
				}
				items = append(items, item)
			}
			out[key] = items
		default:
			out[key] = typed
		}
	}
	return out
}

func normalizeConfig(sourcePaths []string, raw map[string]any) model.ConfigModel {
	cfg := model.ConfigModel{
		SourcePaths: sourcePaths,
		Raw:         raw,
		Receivers:   sectionAsComponents(raw["receivers"], "receiver"),
		Processors:  sectionAsComponents(raw["processors"], "processor"),
		Exporters:   sectionAsComponents(raw["exporters"], "exporter"),
		Connectors:  sectionAsComponents(raw["connectors"], "connector"),
		Extensions:  sectionAsComponents(raw["extensions"], "extension"),
		Pipelines:   map[string]model.PipelineModel{},
	}
	if service, ok := raw["service"].(map[string]any); ok {
		if pipelines, ok := service["pipelines"].(map[string]any); ok {
			for name, item := range pipelines {
				pipeMap, _ := item.(map[string]any)
				cfg.Pipelines[name] = model.PipelineModel{
					Name:       name,
					Signal:     model.DetectSignalType(name),
					Receivers:  stringSlice(pipeMap["receivers"]),
					Processors: stringSlice(pipeMap["processors"]),
					Exporters:  stringSlice(pipeMap["exporters"]),
				}
			}
		}
	}
	yamlText, _ := model.MarshalYAML(raw)
	cfg.CanonicalYAML = yamlText
	return cfg
}

func sectionAsComponents(value any, kind string) map[string]model.Component {
	out := map[string]model.Component{}
	section, _ := value.(map[string]any)
	for name, item := range section {
		itemMap, _ := item.(map[string]any)
		out[name] = model.Component{Name: name, Kind: kind, Config: itemMap}
	}
	return out
}

func stringSlice(value any) []string {
	items, _ := value.([]any)
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, toString(item))
	}
	return out
}

func toString(v any) string {
	switch typed := v.(type) {
	case string:
		return typed
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func dedupeEnv(cfg model.ConfigModel) model.ConfigModel {
	refSeen := map[string]model.EnvReference{}
	for _, ref := range cfg.EnvReferences {
		refSeen[ref.Original+"#"+ref.Name] = ref
	}
	cfg.EnvReferences = make([]model.EnvReference, 0, len(refSeen))
	for _, ref := range refSeen {
		cfg.EnvReferences = append(cfg.EnvReferences, ref)
	}
	sort.Slice(cfg.EnvReferences, func(i, j int) bool {
		return cfg.EnvReferences[i].Name < cfg.EnvReferences[j].Name
	})
	missingSeen := map[string]struct{}{}
	cfg.MissingEnvNames = cfg.MissingEnvNames[:0]
	for _, name := range cfg.EnvReferences {
		if name.HasValue {
			continue
		}
		if _, ok := missingSeen[name.Name]; ok {
			continue
		}
		missingSeen[name.Name] = struct{}{}
		cfg.MissingEnvNames = append(cfg.MissingEnvNames, name.Name)
	}
	sort.Strings(cfg.MissingEnvNames)
	return cfg
}
