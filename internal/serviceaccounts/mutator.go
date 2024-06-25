package serviceaccounts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/snorwin/jsonpatch"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

var __ACCESS_INJECTION_ANNOTATION = `beam.orbitops.dev/access`

type ServiceAccountMutator struct {
	cli dynamic.NamespaceableResourceInterface
}

func NewServiceAccountMutator() *ServiceAccountMutator {
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
		Resource: "externalsecretaccesses",
	}

	return &ServiceAccountMutator{
		cli: dynamicClient.Resource(gvr),
	}
}

func (sam *ServiceAccountMutator) HandleMutate(ctx context.Context, raw []byte) ([]byte, error) {
	// Unmarshal serviceAccount
	sa := &v1.ServiceAccount{}
	original := &v1.ServiceAccount{}
	if err := json.Unmarshal(raw, sa); err != nil {
		return nil, fmt.Errorf("could not unmarshal pod object: %v", err)
	}
	if err := json.Unmarshal(raw, original); err != nil {
		return nil, fmt.Errorf("could not unmarshal serviceAccount object: %v", err)
	}

	var accessName string
	if val, ok := sa.Annotations[__ACCESS_INJECTION_ANNOTATION]; !ok {
		return nil, nil
	} else {
		accessName = val
	}

	secretAccess, err := sam.cli.Namespace(sa.Namespace).Get(ctx, accessName, metav1.GetOptions{})
	if err != nil {
		fmt.Printf("not found: %v\n", err)
		return nil, fmt.Errorf("finding secret access: %w", err)
	}

	if _, ok := secretAccess.Object["status"]; !ok {
		fmt.Println("NO STATUS")
		return nil, fmt.Errorf("no status in object")
	}

	if annotation, ok := (secretAccess.Object["status"]).(map[string]interface{})["provider"].(map[string]interface{})["ServiceAccountAnnotation"].(string); ok {
		splitAnnotation := strings.Split(annotation, "=")
		sa.Annotations[splitAnnotation[0]] = splitAnnotation[1]
	}

	patch, err := jsonpatch.CreateJSONPatch(sa, original)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal serviceAccount object: %v", err)
	}

	return patch.Raw(), nil
}
