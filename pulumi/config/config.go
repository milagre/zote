// Package config loads and merges the per-environment YAML configuration
// tree that downstream components consume. The load algorithm is: walk
// env-specific folders in order, glob for [0-9].*.yaml, render
// ${env.<field>} placeholders, and deep-merge.
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

// fileGlob matches configuration files at each env tier: a single digit,
// a dot, any name, and a .yaml suffix. The leading digit is used purely as
// a lexicographic merge-order hint (lower digit wins base, higher digits
// layer on).
var fileGlob = regexp.MustCompile(`^[0-9]\..*\.yaml$`)

// Config is the output of loading a single environment's config tree.
type Config struct {
	// Sources are the paths (relative to the root) of every YAML file that
	// contributed, in merge order. Kept so callers can surface provenance.
	Sources []string

	// Raw is the per-file decoded YAML, in the same order as Sources.
	Raw []map[string]any

	// Data is the deep-merged union of Raw.
	Data map[string]any

	// EnvVars is pass-through data the caller surfaces alongside the
	// merged YAML — typically the ambient WM_* environment variables every
	// workload inherits.
	EnvVars map[string]string
}

// Loader reads env-specific YAML files from a filesystem and merges them.
type Loader struct {
	// FS is the filesystem to read from. Nil means os.DirFS(Env.Root).
	FS fs.FS
	// Env selects which folders are read.
	Env env.Env
	// EnvVars is pass-through data exposed in the output.
	EnvVars map[string]string
}

// Load reads the tier folders in order, renders templates, and deep-merges.
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
		EnvVars: l.EnvVars,
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

// deepMerge merges src into dst in place. Maps merge recursively, with the
// later source (src) winning for scalar and list values. Fixture-driven
// tests pin the exact semantics we care about.
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
