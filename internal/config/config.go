package config

var defaults = map[string]string{
	"pot_host": "localhost",
	"pot_port": "2222",
}

func GetValue(name string) string {
	// Check if the key exists in the map

	// If not, check in the defaults map
	if val, ok := defaults[name]; ok {
		return val
	}

	return ""
}
