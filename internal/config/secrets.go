package config

type SecretConfig struct {
	Target   string `json:"target"`
	Template string `json:"template"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
}
