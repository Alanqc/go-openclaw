package llm

import "sync"

var (
	plugins   = make(map[ProviderID]Plugin)
	pluginsMu sync.RWMutex
)

// Register 注册一个 LLM 插件。
func Register(p Plugin) {
	pluginsMu.Lock()
	defer pluginsMu.Unlock()
	plugins[p.ID()] = p
}

// Get 按 id 返回已注册的插件，未找到返回 nil。
func Get(id ProviderID) Plugin {
	pluginsMu.RLock()
	defer pluginsMu.RUnlock()
	return plugins[id]
}

// List 返回所有已注册的 LLM 插件。
func List() []Plugin {
	pluginsMu.RLock()
	defer pluginsMu.RUnlock()
	out := make([]Plugin, 0, len(plugins))
	for _, p := range plugins {
		out = append(out, p)
	}
	return out
}
