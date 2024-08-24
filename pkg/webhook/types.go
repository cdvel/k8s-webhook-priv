package webhook

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// ValidationResult represents the result of validation
type ValidationResult struct {
	Allowed bool
	Message string
}

// PodValidator validates pods
type PodValidator struct{}

// ValidatePod validates a pod against security policies
func (v *PodValidator) ValidatePod(pod *corev1.Pod) *ValidationResult {
	// Check security context
	if err := v.checkSecurityContext(pod); err != nil {
		return &ValidationResult{
			Allowed: false,
			Message: err.Error(),
		}
	}

	// Check resources
	if err := v.checkResources(pod); err != nil {
		return &ValidationResult{
			Allowed: false,
			Message: err.Error(),
		}
	}

	// Check image sources
	if err := v.checkImageSources(pod); err != nil {
		return &ValidationResult{
			Allowed: false,
			Message: err.Error(),
		}
	}

	// Check for forbidden volume types
	if err := v.checkVolumes(pod); err != nil {
		return &ValidationResult{
			Allowed: false,
			Message: err.Error(),
		}
	}

	return &ValidationResult{
		Allowed: true,
		Message: "",
	}
}

// checkSecurityContext validates security context settings
func (v *PodValidator) checkSecurityContext(pod *corev1.Pod) error {
	// Check if pod runs as non-root
	if pod.Spec.SecurityContext != nil {
		if pod.Spec.SecurityContext.RunAsNonRoot != nil && !*pod.Spec.SecurityContext.RunAsNonRoot {
			return &ValidationError{Message: "Pod must run as non-root user"}
		}
		if pod.Spec.SecurityContext.RunAsUser != nil && *pod.Spec.SecurityContext.RunAsUser == 0 {
			return &ValidationError{Message: "Pod cannot run as root user (UID 0)"}
		}
	}

	// Check each container
	for _, container := range pod.Spec.Containers {
		if container.SecurityContext != nil {
			// Check if container runs as non-root
			if container.SecurityContext.RunAsNonRoot != nil && !*container.SecurityContext.RunAsNonRoot {
				return &ValidationError{Message: "Container must run as non-root user"}
			}
			if container.SecurityContext.RunAsUser != nil && *container.SecurityContext.RunAsUser == 0 {
				return &ValidationError{Message: "Container cannot run as root user (UID 0)"}
			}

			// Check if privileged
			if container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
				return &ValidationError{Message: "Container cannot be privileged"}
			}

			// Check if read-only root filesystem
			if container.SecurityContext.ReadOnlyRootFilesystem != nil && !*container.SecurityContext.ReadOnlyRootFilesystem {
				return &ValidationError{Message: "Container must have readOnlyRootFilesystem=true"}
			}
		}
	}

	return nil
}

// checkResources validates resource limits and requests
func (v *PodValidator) checkResources(pod *corev1.Pod) error {
	for _, container := range pod.Spec.Containers {
		// Check if resources are defined
		if container.Resources.Limits == nil || container.Resources.Requests == nil {
			return &ValidationError{Message: "All containers must have resource limits and requests defined"}
		}

		// Check if CPU and memory limits are defined
		cpuLimit := container.Resources.Limits.Cpu()
		memLimit := container.Resources.Limits.Memory()

		if cpuLimit == nil || memLimit == nil {
			return &ValidationError{Message: "All containers must have CPU and memory limits defined"}
		}

		cpuRequest := container.Resources.Requests.Cpu()
		memRequest := container.Resources.Requests.Memory()

		if cpuRequest == nil || memRequest == nil {
			return &ValidationError{Message: "All containers must have CPU and memory requests defined"}
		}
	}

	return nil
}

// checkImageSources validates that images come from approved registries
func (v *PodValidator) checkImageSources(pod *corev1.Pod) error {
	approvedRegistries := []string{
		"gcr.io",
		"docker.io",
		"quay.io",
		"ghcr.io",
	}

	for _, container := range pod.Spec.Containers {
		isApproved := false
		for _, registry := range approvedRegistries {
			if strings.HasPrefix(container.Image, registry+"/") || container.Image == registry {
				isApproved = true
				break
			}
		}

		if !isApproved {
			return &ValidationError{Message: "Container image from unapproved registry: " + container.Image}
		}
	}

	return nil
}

// checkVolumes validates volume types
func (v *PodValidator) checkVolumes(pod *corev1.Pod) error {
	for _, volume := range pod.Spec.Volumes {
		// Check for forbidden volume types
		if volume.HostPath != nil {
			return &ValidationError{Message: "HostPath volumes are not allowed"}
		}

		if volume.EmptyDir != nil && volume.EmptyDir.SizeLimit != nil {
			// Check if EmptyDir size is too large
		}
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
// Validation logic
// Validation logic
