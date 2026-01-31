package channels

import "sync"

var (
	plugins   = make(map[ChannelId]ChannelPlugin)
	pluginsMu sync.RWMutex
)

// Register adds a channel plugin.
func Register(p ChannelPlugin) {
	pluginsMu.Lock()
	defer pluginsMu.Unlock()
	plugins[p.ID()] = p
}

// Get returns a plugin by id.
func Get(id ChannelId) ChannelPlugin {
	pluginsMu.RLock()
	defer pluginsMu.RUnlock()
	return plugins[id]
}

// List returns all registered plugins.
func List() []ChannelPlugin {
	pluginsMu.RLock()
	defer pluginsMu.RUnlock()
	out := make([]ChannelPlugin, 0, len(plugins))
	for _, p := range plugins {
		out = append(out, p)
	}
	return out
}
