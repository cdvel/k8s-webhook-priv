package webhook

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/mattbaird/jsonpatch"
)

// MutateHandler handles mutation requests
func (wh *WebhookServer) MutateHandler(w http.ResponseWriter, r *http.Request) {
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
		response := wh.processMutation(admissionReviewReq.Request)
		admissionReviewResp.Response = response
		admissionReviewResp.Response.UID = admissionReviewReq.Request.UID
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(admissionReviewResp); err != nil {
		log.Printf("Failed to encode response: %v", err)
	}
}

// processMutation processes the mutation request
func (wh *WebhookServer) processMutation(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	log.Printf("Processing mutation for %s/%s in namespace %s", req.Resource.Resource, req.Name, req.Namespace)

	// Skip mutation for certain namespaces
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

	// Perform mutation
	patches, err := wh.mutatePod(&pod)
	if err != nil {
		log.Printf("Failed to mutate pod: %v", err)
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("Failed to mutate pod: %v", err),
			},
		}
	}

	// Convert patches to JSON
	patchData, err := json.Marshal(patches)
	if err != nil {
		log.Printf("Failed to marshal patches: %v", err)
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Message: fmt.Sprintf("Failed to marshal patches: %v", err),
			},
		}
	}

	patchType := admissionv1.PatchTypeJSONPatch
	response := &admissionv1.AdmissionResponse{
		Allowed: true,
		PatchType: &patchType,
		Patch:     patchData,
	}

	log.Printf("Applied %d patches to pod %s/%s", len(patches), req.Namespace, req.Name)

	return response
}

// mutatePod applies mutations to a pod
func (wh *WebhookServer) mutatePod(pod *corev1.Pod) ([]jsonpatch.JsonPatchOperation, error) {
	var patches []jsonpatch.JsonPatchOperation

	// Inject standard labels if they don't exist
	patches = append(patches, wh.injectLabels(pod)...)

	// Inject standard annotations
	patches = append(patches, wh.injectAnnotations(pod)...)

	// Inject default resource limits if missing
	patches = append(patches, wh.injectResourceLimits(pod)...)

	// Inject security context if missing
	patches = append(patches, wh.injectSecurityContext(pod)...)

	return patches, nil
}

// injectLabels injects standard labels
func (wh *WebhookServer) injectLabels(pod *corev1.Pod) []jsonpatch.JsonPatchOperation {
	var patches []jsonpatch.JsonPatchOperation

	if pod.Labels == nil {
		patches = append(patches, jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/metadata/labels",
			Value:     map[string]string{},
		})
	}

	// Add managed-by label if not present
	if _, exists := pod.Labels["managed-by"]; !exists {
		patches = append(patches, jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/metadata/labels/managed-by",
			Value:     "admission-webhook",
		})
	}

	// Add security-level label if not present
	if _, exists := pod.Labels["security-level"]; !exists {
		patches = append(patches, jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/metadata/labels/security-level",
			Value:     "standard",
		})
	}

	return patches
}

// injectAnnotations injects standard annotations
func (wh *WebhookServer) injectAnnotations(pod *corev1.Pod) []jsonpatch.JsonPatchOperation {
	var patches []jsonpatch.JsonPatchOperation

	if pod.Annotations == nil {
		patches = append(patches, jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/metadata/annotations",
			Value:     map[string]string{},
		})
	}

	// Add admission timestamp annotation
	timestamp := metav1.Now().Format("2006-01-02T15:04:05Z")
	patches = append(patches, jsonpatch.JsonPatchOperation{
		Operation: "add",
		Path:      "/metadata/annotations/admission.webhook.io~1admitted-at",
		Value:     timestamp,
	})

	// Add webhook version annotation
	patches = append(patches, jsonpatch.JsonPatchOperation{
		Operation: "add",
		Path:      "/metadata/annotations/admission.webhook.io~1version",
		Value:     "v1.0.0",
	})

	return patches
}

// injectResourceLimits injects default resource limits if missing
func (wh *WebhookServer) injectResourceLimits(pod *corev1.Pod) []jsonpatch.JsonPatchOperation {
	var patches []jsonpatch.JsonPatchOperation

	defaultCpuLimit := "500m"
	defaultMemLimit := "512Mi"
	defaultCpuRequest := "100m"
	defaultMemRequest := "128Mi"

	for i, container := range pod.Spec.Containers {
		needsUpdate := false
		containerPath := fmt.Sprintf("/spec/containers/%d", i)

		// Check if resources are defined
		if container.Resources.Limits == nil {
			container.Resources.Limits = corev1.ResourceList{}
			needsUpdate = true
		}
		if container.Resources.Requests == nil {
			container.Resources.Requests = corev1.ResourceList{}
			needsUpdate = true
		}

		// Add default CPU limit if missing
		if container.Resources.Limits.Cpu().IsZero() {
			container.Resources.Limits[corev1.ResourceCPU] = resource.MustParse(defaultCpuLimit)
			needsUpdate = true
		}

		// Add default memory limit if missing
		if container.Resources.Limits.Memory().IsZero() {
			container.Resources.Limits[corev1.ResourceMemory] = resource.MustParse(defaultMemLimit)
			needsUpdate = true
		}

		// Add default CPU request if missing
		if container.Resources.Requests.Cpu().IsZero() {
			container.Resources.Requests[corev1.ResourceCPU] = resource.MustParse(defaultCpuRequest)
			needsUpdate = true
		}

		// Add default memory request if missing
		if container.Resources.Requests.Memory().IsZero() {
			container.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(defaultMemRequest)
			needsUpdate = true
		}

		if needsUpdate {
			resourcesPath := containerPath + "/resources"
			patches = append(patches, jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      resourcesPath,
				Value:     container.Resources,
			})
		}
	}

	return patches
}

// injectSecurityContext injects default security context if missing
func (wh *WebhookServer) injectSecurityContext(pod *corev1.Pod) []jsonpatch.JsonPatchOperation {
	var patches []jsonpatch.JsonPatchOperation

	// Add pod security context if missing
	if pod.Spec.SecurityContext == nil {
		runAsNonRoot := true
		fsGroup := int64(1000)

		patches = append(patches, jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/spec/securityContext",
			Value: &corev1.PodSecurityContext{
				RunAsNonRoot: &runAsNonRoot,
				FSGroup:      &fsGroup,
			},
		})
	}

	// Add security context to containers if missing
	for i, container := range pod.Spec.Containers {
		containerPath := fmt.Sprintf("/spec/containers/%d", i)
		needsUpdate := false

		if container.SecurityContext == nil {
			container.SecurityContext = &corev1.SecurityContext{}
			needsUpdate = true
		}

		if container.SecurityContext.RunAsNonRoot == nil {
			runAsNonRoot := true
			container.SecurityContext.RunAsNonRoot = &runAsNonRoot
			needsUpdate = true
		}

		if container.SecurityContext.ReadOnlyRootFilesystem == nil {
			readOnlyRootFilesystem := true
			container.SecurityContext.ReadOnlyRootFilesystem = &readOnlyRootFilesystem
			needsUpdate = true
		}

		if container.SecurityContext.AllowPrivilegeEscalation == nil {
			allowPrivilegeEscalation := false
			container.SecurityContext.AllowPrivilegeEscalation = &allowPrivilegeEscalation
			needsUpdate = true
		}

		if needsUpdate {
			securityContextPath := containerPath + "/securityContext"
			patches = append(patches, jsonpatch.JsonPatchOperation{
				Operation: "add",
				Path:      securityContextPath,
				Value:     container.SecurityContext,
			})
		}
	}

	return patches
}
// Mutation functionality
