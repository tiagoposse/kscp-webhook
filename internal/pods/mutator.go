package pods

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/snorwin/jsonpatch"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

type PodMutator struct {
	cli dynamic.NamespaceableResourceInterface
}

func NewPodMutator() *PodMutator {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create in-cluster config: %v", err)
	}

	// Create a dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
	}

	// Define the GVR (Group-Version-Resource) for the SecretAccess custom resource
	gvr := schema.GroupVersionResource{
		Group:    "orbitops.dev",
		Version:  "v1alpha1",
		Resource: "externalsecrets",
	}

	return &PodMutator{
		cli: dynamicClient.Resource(gvr),
	}
}

func (pm *PodMutator) HandleMutate(ctx context.Context, raw []byte) ([]byte, error) {
	// Unmarshal pods
	pod := &v1.Pod{}
	original := &v1.Pod{}
	if err := json.Unmarshal(raw, pod); err != nil {
		return nil, fmt.Errorf("could not unmarshal pod object: %v", err)
	}
	if err := json.Unmarshal(raw, original); err != nil {
		return nil, fmt.Errorf("could not unmarshal pod object: %v", err)
	}

	if _, ok := pod.Annotations[__SKIP_ANNOTATION]; ok {
		return []byte{}, nil
	}

	// Get secrets from annotations
	secrets, err := pm.extractSecrets(ctx, pod)
	if err != nil {
		return nil, fmt.Errorf("extracting secrets: %w", err)
	}

	paths := make([]string, 0)
	if pod.Spec.InitContainers == nil {
		pod.Spec.InitContainers = make([]v1.Container, 0)
	}

	for provider, providerSecrets := range secrets {
		cont, err := pm.parseProvider(ctx, provider, pod, providerSecrets)
		if err != nil {
			return nil, fmt.Errorf("could not unmarshal pod object: %v", err)
		}

		paths = MutateVolumes(provider, cont.VolumeMounts, pod, paths)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *cont)
	}

	pod.Annotations[__SKIP_ANNOTATION] = "true"

	patch, err := jsonpatch.CreateJSONPatch(pod, original)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal pod object: %v", err)
	}

	return patch.Raw(), nil
}
