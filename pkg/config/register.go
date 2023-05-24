package config

var extraConfigs = map[string]func() Configuration{}

func RegisterConfig(key string, f func() Configuration) {
	extraConfigs[key] = f
}

func ResolveConfig() map[string]func() Configuration {
	return extraConfigs
}
