package mutator

import (
	"regexp"

	"github.com/tiagoposse/kscp-webhook/internal/config"
)

var __INJECTION_ANNOTATION = `secrets.kscp.io/secret-(\w+)`
var __INJECTION_TEMPLATE_ANNOTATION = `secrets.kscp.io/secret-{NAME}-template`
var __INJECTION_TARGET_ANNOTATION = `secrets.kscp.io/secret-{NAME}-target`
var __INJECTION_POSITION_ANNOTATION = `secrets.kscp.io/container-position`
var __SKIP_ANNOTATION = `secrets.kscp.io/injected`

func GetSecretsFromAnnotations(podAnnotations map[string]string) map[string]map[string]*config.SecretConfig {
	// { provider: { secret: { key: value }}}
	secrets := make(map[string]map[string]*config.SecretConfig)
	nameRe := regexp.MustCompile(__INJECTION_ANNOTATION)

	// Check for annotations starting with "kscp.io"
	for key := range podAnnotations {
		if matches := nameRe.FindStringSubmatch(key); len(matches) > 0 {
			provider := podAnnotations[key]
			if _, ok := secrets[provider]; !ok {
				secrets[provider] = make(map[string]*config.SecretConfig)
			}

			secrets[provider][matches[1]] = &config.SecretConfig{}
		}
	}

	return secrets
}
