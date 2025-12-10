package zstatscmd

import (
	"encoding/json"
	"fmt"

	"github.com/milagre/zote/go/zcmd"
	"github.com/milagre/zote/go/zstats"
)

var _ zcmd.Aspect = Aspect{}

type Aspect struct {
	globalPrefix string
}

func NewStats(globalPrefix string) Aspect {
	return Aspect{
		globalPrefix: globalPrefix,
	}
}

func (a Aspect) Apply(c zcmd.Configurable) {
	c.AddString("stats-prefix")
	c.AddString("stats-tags")
}

func (a Aspect) Configure(env zcmd.Env, stats zstats.Stats) (zstats.Stats, error) {
	prefix := env.String("stats-prefix")
	if prefix != "" {
		stats.AddPrefix(prefix)
	}

	tags := env.String("stats-tags")
	if tags != "" {
		var tagsMap zstats.Tags
		err := json.Unmarshal([]byte(tags), &tagsMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal stats tags: %w", err)
		}
		stats.AddTags(tagsMap)
	}

	return stats, nil
}
