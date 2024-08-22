package webhook

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	runtimeScheme = runtime.NewScheme()
	codecs        = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecs.UniversalDeserializer()
)

func init() {
	_ = corev1.AddToScheme(runtimeScheme)
}

// ValidateHandler handles validation requests
func (wh *WebhookServer) ValidateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	body := make([]byte, r.ContentLength)
	_, err := r.Body.Read(body)
	if err != nil && err.Error() != "EOF" {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var admissionReviewReq admissionv1.AdmissionReview
	_, _, err = deserializer.Decode(body, nil, &admissionReviewReq)
	if err != nil {
		log.Printf("Failed to decode admission review: %v", err)
		http.Error(w, "Failed to decode admission review", http.StatusBadRequest)
		return
	}

	var admissionReviewResp admissionv1.AdmissionReview
	admissionReviewResp.APIVersion = "admission.k8s.io/v1"
	admissionReviewResp.Kind = "AdmissionReview"

	if admissionReviewReq.Request != nil {
		response := wh.processValidation(admissionReviewReq.Request)
		admissionReviewResp.Response = response
		admissionReviewResp.Response.UID = admissionReviewReq.Request.UID
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(admissionReviewResp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// processValidation processes the validation request
func (wh *WebhookServer) processValidation(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	log.Printf("Processing validation for %s/%s in namespace %s", req.Resource.Resource, req.Name, req.Namespace)

	// Skip validation for certain namespaces
	if req.Namespace == "kube-system" || req.Namespace == "kube-public" || req.Namespace == "webhook-system" {
		return &admissionv1.AdmissionResponse{
			Allowed: true,
		}
	}

	// Decode the pod object
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		log.Printf("Could not unmarshal pod: %v", err)
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("Could not unmarshal pod: %v", err),
			},
		}
	}

	// Perform validation
	validator := &PodValidator{}
	result := validator.ValidatePod(&pod)

	// Log the validation result
	log.Printf("Validation result for pod %s/%s: allowed=%t, reason=%s", req.Namespace, req.Name, result.Allowed, result.Message)

	admissionResponse := &admissionv1.AdmissionResponse{
		Allowed: result.Allowed,
	}

	if !result.Allowed {
		admissionResponse.Result = &metav1.Status{
			Message: result.Message,
		}
	}

	return admissionResponse
}

// HealthHandler handles health checks
func (wh *WebhookServer) HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}