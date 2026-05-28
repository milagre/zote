// Package config loads merged per-env YAML: walk env/<type>/<tier>/<name>, glob [0-9].*.yaml, template ${env.*}, deep-merge.
package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/milagre/zote/pulumi/env"
)

// Leading digit in filenames sets merge order (see tests).
var fileGlob = regexp.MustCompile(`^[0-9]\..*\.yaml$`)

type Config struct {
	Sources []string
	Raw     []map[string]any
	Data    map[string]any
}

type Loader struct {
	FS  fs.FS // nil → os.DirFS(Env.Root)
	Env env.Env
}

func (l *Loader) Load() (*Config, error) {
	if err := l.Env.Validate(); err != nil {
		return nil, fmt.Errorf("invalid env: %w", err)
	}

	fsys := l.FS
	if fsys == nil {
		fsys = os.DirFS(l.Env.Root)
	}

	folders := []string{
		path.Join("env", l.Env.Type),
		path.Join("env", l.Env.Type, l.Env.Tier),
		path.Join("env", l.Env.Type, l.Env.Tier, l.Env.Name),
	}

	var (
		sources []string
		raw     []map[string]any
	)
	for _, folder := range folders {
		files, err := listConfigFiles(fsys, folder)
		if err != nil {
			return nil, fmt.Errorf("listing %s: %w", folder, err)
		}

		for _, file := range files {
			full := path.Join(folder, file)
			data, err := fs.ReadFile(fsys, full)
			if err != nil {
				return nil, fmt.Errorf("reading %s: %w", full, err)
			}

			rendered, err := renderTemplate(string(data), l.Env)
			if err != nil {
				return nil, fmt.Errorf("templating %s: %w", full, err)
			}

			var decoded map[string]any
			if err := yaml.Unmarshal([]byte(rendered), &decoded); err != nil {
				return nil, fmt.Errorf("yaml %s: %w", full, err)
			}
			if decoded == nil {
				decoded = map[string]any{}
			}

			sources = append(sources, full)
			raw = append(raw, decoded)
		}
	}

	merged := map[string]any{}
	for _, m := range raw {
		deepMerge(merged, m)
	}

	return &Config{
		Sources: sources,
		Raw:     raw,
		Data:    merged,
	}, nil
}

func listConfigFiles(fsys fs.FS, folder string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, folder)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}

		return nil, err
	}

	var out []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !fileGlob.MatchString(e.Name()) {
			continue
		}
		out = append(out, e.Name())
	}
	sort.Strings(out)

	return out, nil
}

func deepMerge(dst, src map[string]any) {
	for k, srcVal := range src {
		dstVal, ok := dst[k]
		if !ok {
			dst[k] = srcVal

			continue
		}

		dstMap, dstIsMap := dstVal.(map[string]any)
		srcMap, srcIsMap := srcVal.(map[string]any)
		if dstIsMap && srcIsMap {
			deepMerge(dstMap, srcMap)

			continue
		}

		dst[k] = srcVal
	}
}
