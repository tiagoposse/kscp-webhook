package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/tiagoposse/secretsbeam-webhook/internal/pods"
	"github.com/tiagoposse/secretsbeam-webhook/internal/serviceaccounts"
	admissionv1 "k8s.io/api/admission/v1"
)

type server struct {
	port        int
	tlsCertPath string
	tlsKeyPath  string
	sslEnabled  bool
	pod         *pods.PodMutator
	sa          *serviceaccounts.ServiceAccountMutator
}

type Mutator interface {
	HandleMutate(context.Context, []byte) ([]byte, error)
}

func NewServer(opts ...ServerOption) *server {
	sa := serviceaccounts.NewServiceAccountMutator()
	pod := pods.NewPodMutator()

	http.HandleFunc("/healthz", health)
	http.HandleFunc("/serviceaccounts", func(w http.ResponseWriter, r *http.Request) {
		handleMutate(w, r, sa)
	})
	http.HandleFunc("/pods", func(w http.ResponseWriter, r *http.Request) {
		handleMutate(w, r, pod)
	})

	s := &server{
		sa:  sa,
		pod: pod,
	}
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

func handleMutate(w http.ResponseWriter, r *http.Request, mut Mutator) {
	var admissionReview admissionv1.AdmissionReview

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

	patch, err := mut.HandleMutate(context.Background(), admissionReview.Request.Object.Raw)
	if err != nil {
		fmt.Printf("mutating: %v\n", err.Error())
		http.Error(w, fmt.Sprintf("executing mutation: %v", err), http.StatusInternalServerError)
		return
	}

	admissionReview.Response = &admissionv1.AdmissionResponse{
		UID:     admissionReview.Request.UID,
		Allowed: true,
		Patch:   patch,
		PatchType: func() *admissionv1.PatchType {
			pt := admissionv1.PatchTypeJSONPatch
			return &pt
		}(),
	}

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
