package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/snorwin/jsonpatch"
	"github.com/tiagoposse/kscp-webhook/internal/mutator"
	admissionv1 "k8s.io/api/admission/v1"
	v1 "k8s.io/api/core/v1"
)

type server struct {
	port        int
	tlsCertPath string
	tlsKeyPath  string
	sslEnabled  bool
}

func NewServer(opts ...ServerOption) *server {
	http.HandleFunc("/healthz", health)
	http.HandleFunc("/mutate", handleMutate)

	s := &server{}
	for _, o := range opts {
		o(s)
	}

	if s.tlsCertPath != "" && s.tlsKeyPath != "" {
		s.sslEnabled = true
	}

	return s
}

func (s *server) Serve() error {
	fmt.Printf("Starting server on :%d\n", s.port)
	if err := http.ListenAndServeTLS(":443", os.Getenv("CERT_FILE_PATH"), os.Getenv("CERT_KEY_PATH"), nil); err != nil {
		return fmt.Errorf("failed to listen and serve: %v", err)
	}

	return nil
}

func handleMutate(w http.ResponseWriter, r *http.Request) {
	var admissionReview admissionv1.AdmissionReview
	var admissionResponse admissionv1.AdmissionResponse

	// Unmarshal whole request
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not read request body: %v", err), http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(body, &admissionReview); err != nil {
		http.Error(w, fmt.Sprintf("could not unmarshal request: %v", err), http.StatusBadRequest)
		return
	}

	// Unmarshal pods
	pod := &v1.Pod{}
	original := &v1.Pod{}
	if err := json.Unmarshal(admissionReview.Request.Object.Raw, pod); err != nil {
		http.Error(w, fmt.Sprintf("could not unmarshal pod object: %v", err), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(admissionReview.Request.Object.Raw, original); err != nil {
		http.Error(w, fmt.Sprintf("could not unmarshal pod object: %v", err), http.StatusBadRequest)
		return
	}

	// Get secrets from annotations
	secrets := mutator.GetSecretsFromAnnotations(pod.Annotations)

	paths := make([]string, 0)
	if pod.Spec.InitContainers == nil {
		pod.Spec.InitContainers = make([]v1.Container, 0)
	}
	for provider, providerSecrets := range secrets {
		cont, err := mutator.ParseProvider(provider, pod, providerSecrets)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not unmarshal pod object: %v", err), http.StatusBadRequest)
			return
		}

		paths = mutator.MutateVolumes(provider, cont.VolumeMounts, pod, paths)
		pod.Spec.InitContainers = append(pod.Spec.InitContainers, *cont)
	}

	patch, err := jsonpatch.CreateJSONPatch(pod, original)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not unmarshal pod object: %v", err), http.StatusBadRequest)
		return
	}

	admissionResponse = admissionv1.AdmissionResponse{
		UID:     admissionReview.Request.UID,
		Patch:   patch.Raw(),
		Allowed: true,
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch
			return &pt
		}(),
	}

	admissionReview.Response = &admissionResponse
	respBytes, err := json.Marshal(admissionReview)
	if err != nil {
		http.Error(w, fmt.Sprintf("could not marshal response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
