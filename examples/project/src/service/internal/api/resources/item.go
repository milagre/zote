package resources

import (
	"time"

	"github.com/milagre/zote/examples/project/src/service/internal/models"
)

// Item is the JSON resource for item endpoints (not the persistence model).
type Item struct {
	ID       string     `json:"id"`
	Created  time.Time  `json:"created"`
	Modified *time.Time `json:"modified,omitempty"`
	Name     string     `json:"name"`
}

// ItemFromModel maps a stored row to the API resource.
func ItemFromModel(m *models.Item) Item {
	if m == nil {
		return Item{}
	}
	return Item{
		ID:       m.ID,
		Created:  m.Created,
		Modified: m.Modified,
		Name:     m.Name,
	}
}

// ItemListFromModels maps find results to API resources.
func ItemListFromModels(rows []*models.Item) []Item {
	out := make([]Item, 0, len(rows))
	for _, r := range rows {
		if r == nil {
			continue
		}
		out = append(out, ItemFromModel(r))
	}
	return out
}
