package pods

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/tiagoposse/secretsbeam-webhook/internal/config"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var __INJECTION_ANNOTATION = `beam.orbitops.dev/secret-([\w\d-_]+)`
var __INJECTION_TEMPLATE_ANNOTATION = `beam.orbitops.dev/secret-{NAME}-template`
var __INJECTION_NAMESPACE_ANNOTATION = `beam.orbitops.dev/secret-{NAME}-namespace`
var __INJECTION_TARGET_ANNOTATION = `beam.orbitops.dev/secret-{NAME}-target`
var __INJECTION_POSITION_ANNOTATION = `beam.orbitops.dev/container-position`
var __SKIP_ANNOTATION = `beam.orbitops.dev/injected`

func (m *PodMutator) extractSecrets(ctx context.Context, pod *v1.Pod) (map[string][]*config.SecretConfig, error) {
	// { provider: { secret: { key: value }}}
	secrets := make(map[string][]*config.SecretConfig)
	nameRe := regexp.MustCompile(__INJECTION_ANNOTATION)

	// Check for annotations starting with "orbitops.dev"
	for key := range pod.Annotations {
		if matches := nameRe.FindStringSubmatch(key); len(matches) > 0 {
			cfg, err := m.extractSecretConfig(ctx, matches[1], pod.Namespace, pod.Annotations)
			if err != nil {
				return nil, fmt.Errorf("extracting config: %w", err)
			}

			if _, ok := secrets[cfg.Provider]; !ok {
				secrets[cfg.Provider] = make([]*config.SecretConfig, 0)
			}

			secrets[cfg.Provider] = append(secrets[cfg.Provider], cfg)
		}
	}

	return secrets, nil
}

func (m *PodMutator) extractSecretConfig(ctx context.Context, secretName, namespace string, podAnnotations map[string]string) (*config.SecretConfig, error) {
	namespaceAnnotation := strings.ReplaceAll(__INJECTION_NAMESPACE_ANNOTATION, "{NAME}", secretName)
	if val, ok := podAnnotations[namespaceAnnotation]; ok {
		namespace = val
	}

	obj, err := m.cli.Namespace(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("getting secret %s: %w", secretName, err)
	}

	objStatus := obj.Object["status"].(map[string]interface{})
	objSpec := obj.Object["spec"].(map[string]interface{})
	cfg := &config.SecretConfig{
		Name:     objStatus["name"].(string),
		Provider: objSpec["provider"].(string),
	}

	templateAnnotation := strings.ReplaceAll(__INJECTION_TEMPLATE_ANNOTATION, "{NAME}", secretName)
	targetAnnotation := strings.ReplaceAll(__INJECTION_TARGET_ANNOTATION, "{NAME}", secretName)

	if val, ok := podAnnotations[templateAnnotation]; ok {
		cfg.Template = val
	}

	if val, ok := podAnnotations[targetAnnotation]; ok {
		cfg.Target = val
	} else {
		cfg.Target = fmt.Sprintf("/var/run/secrets/orbitops.dev/%s", secretName)
	}

	return cfg, nil
}
