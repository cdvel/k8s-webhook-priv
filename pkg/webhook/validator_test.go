package webhook

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestValidatePodSecurityContext(t *testing.T) {
	validator := &PodValidator{}

	// Test case 1: Pod with RunAsNonRoot = true should pass
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: &[]bool{true}[0],
			},
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "docker.io/nginx:latest",
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
			},
		},
	}

	result1 := validator.ValidatePod(pod1)
	if !result1.Allowed {
		t.Errorf("Expected pod with RunAsNonRoot=true to be allowed, but got: %s", result1.Message)
	}

	// Test case 2: Pod with RunAsNonRoot = false should fail
	runAsNonRootFalse := false
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-2",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: &runAsNonRootFalse,
			},
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "docker.io/nginx:latest",
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
			},
		},
	}

	result2 := validator.ValidatePod(pod2)
	if result2.Allowed {
		t.Errorf("Expected pod with RunAsNonRoot=false to be denied, but it was allowed")
	}
	if result2.Message == "" {
		t.Error("Expected error message for RunAsNonRoot=false validation")
	}
}

func TestValidatePodResources(t *testing.T) {
	validator := &PodValidator{}

	// Test case 1: Pod with proper resource limits and requests should pass
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: &[]bool{true}[0],
			},
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "docker.io/nginx:latest",
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
			},
		},
	}

	result1 := validator.ValidatePod(pod1)
	if !result1.Allowed {
		t.Errorf("Expected pod with proper resources to be allowed, but got: %s", result1.Message)
	}

	// Test case 2: Pod without resource limits should fail
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-2",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: &[]bool{true}[0],
			},
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "docker.io/nginx:latest",
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
					},
					Resources: corev1.ResourceRequirements{
						// No limits or requests
					},
				},
			},
		},
	}

	result2 := validator.ValidatePod(pod2)
	if result2.Allowed {
		t.Errorf("Expected pod without resource limits to be denied, but it was allowed")
	}
	if result2.Message == "" {
		t.Error("Expected error message for missing resources validation")
	}
}

func TestValidatePodImageSources(t *testing.T) {
	validator := &PodValidator{}

	// Test case 1: Pod with approved registry should pass
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: &[]bool{true}[0],
			},
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "docker.io/nginx:latest",
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
			},
		},
	}

	result1 := validator.ValidatePod(pod1)
	if !result1.Allowed {
		t.Errorf("Expected pod with approved registry to be allowed, but got: %s", result1.Message)
	}

	// Test case 2: Pod with unapproved registry should fail
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-2",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: &[]bool{true}[0],
			},
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "myregistry.com/nginx:latest",
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
			},
		},
	}

	result2 := validator.ValidatePod(pod2)
	if result2.Allowed {
		t.Errorf("Expected pod with unapproved registry to be denied, but it was allowed")
	}
	if result2.Message == "" {
		t.Error("Expected error message for unapproved registry validation")
	}
}

func TestValidatePodPrivilegedContainer(t *testing.T) {
	validator := &PodValidator{}

	// Test case 1: Pod with non-privileged container should pass
	privilegedFalse := false
	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-1",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: &[]bool{true}[0],
			},
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "docker.io/nginx:latest",
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						Privileged:   &privilegedFalse,
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
			},
		},
	}

	result1 := validator.ValidatePod(pod1)
	if !result1.Allowed {
		t.Errorf("Expected pod with non-privileged container to be allowed, but got: %s", result1.Message)
	}

	// Test case 2: Pod with privileged container should fail
	privilegedTrue := true
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod-2",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: &[]bool{true}[0],
			},
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "docker.io/nginx:latest",
					SecurityContext: &corev1.SecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						Privileged:   &privilegedTrue,
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
					},
				},
			},
		},
	}

	result2 := validator.ValidatePod(pod2)
	if result2.Allowed {
		t.Errorf("Expected pod with privileged container to be denied, but it was allowed")
	}
	if result2.Message == "" {
		t.Error("Expected error message for privileged container validation")
	}
}
// Tests
