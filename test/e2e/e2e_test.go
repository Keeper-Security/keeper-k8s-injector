// Package e2e contains end-to-end tests for the Keeper K8s Injector.
// These tests require a running Kubernetes cluster (Kind recommended).
//
// Run with: go test -v -tags=e2e ./test/e2e/...
//
//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset *kubernetes.Clientset
	namespace = "keeper-e2e-test"
)

func TestMain(m *testing.M) {
	// Setup
	if err := setup(); err != nil {
		fmt.Printf("Setup failed: %v\n", err)
		os.Exit(1)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	cleanup()

	os.Exit(code)
}

func setup() error {
	// Load kubeconfig
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = os.Getenv("HOME") + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	// Create test namespace
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err = clientset.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	if err != nil {
		// Namespace might already exist
		fmt.Printf("Note: namespace creation returned: %v\n", err)
	}

	return nil
}

func cleanup() {
	if clientset == nil {
		return
	}
	// Delete test namespace
	_ = clientset.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
}

func TestInjectorInstalled(t *testing.T) {
	ctx := context.Background()

	// Check that the injector deployment exists
	deployment, err := clientset.AppsV1().Deployments("keeper-system").Get(ctx, "keeper-injector", metav1.GetOptions{})
	if err != nil {
		t.Skipf("Injector not installed, skipping: %v", err)
		return
	}

	if deployment.Status.ReadyReplicas < 1 {
		t.Errorf("Injector has %d ready replicas, want at least 1", deployment.Status.ReadyReplicas)
	}
}

func TestSimplePodInjection(t *testing.T) {
	ctx := context.Background()

	// Skip if no KSM config
	ksmConfig := os.Getenv("KSM_CONFIG")
	if ksmConfig == "" {
		t.Skip("KSM_CONFIG not set, skipping integration test")
	}

	// Create auth secret
	authSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "keeper-auth-test",
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"config": []byte(ksmConfig),
		},
	}
	_, err := clientset.CoreV1().Secrets(namespace).Create(ctx, authSecret, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create auth secret: %v", err)
	}
	defer clientset.CoreV1().Secrets(namespace).Delete(ctx, "keeper-auth-test", metav1.DeleteOptions{})

	// Create test pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-injection-pod",
			Namespace: namespace,
			Annotations: map[string]string{
				"keeper.security/inject":      "true",
				"keeper.security/auth-secret": "keeper-auth-test",
				"keeper.security/secret":      "test-secret", // Must exist in KSM
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "test",
					Image:   "alpine:latest",
					Command: []string{"sleep", "300"},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	createdPod, err := clientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test pod: %v", err)
	}
	defer clientset.CoreV1().Pods(namespace).Delete(ctx, "test-injection-pod", metav1.DeleteOptions{})

	// Wait for pod to be running
	err = waitForPodRunning(ctx, namespace, "test-injection-pod", 60*time.Second)
	if err != nil {
		t.Fatalf("Pod did not reach Running state: %v", err)
	}

	// Verify init container was injected
	updatedPod, err := clientset.CoreV1().Pods(namespace).Get(ctx, "test-injection-pod", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get updated pod: %v", err)
	}

	hasInitContainer := false
	for _, ic := range updatedPod.Spec.InitContainers {
		if ic.Name == "keeper-secrets-init" {
			hasInitContainer = true
			break
		}
	}
	if !hasInitContainer {
		t.Error("Init container was not injected")
	}

	// Verify sidecar was injected
	hasSidecar := false
	for _, c := range updatedPod.Spec.Containers {
		if c.Name == "keeper-secrets-sidecar" {
			hasSidecar = true
			break
		}
	}
	if !hasSidecar {
		t.Error("Sidecar container was not injected")
	}

	// Verify secrets volume was added
	hasSecretsVolume := false
	for _, v := range updatedPod.Spec.Volumes {
		if v.Name == "keeper-secrets" {
			hasSecretsVolume = true
			if v.EmptyDir == nil || v.EmptyDir.Medium != corev1.StorageMediumMemory {
				t.Error("Secrets volume should be memory-backed")
			}
			break
		}
	}
	if !hasSecretsVolume {
		t.Error("Secrets volume was not added")
	}

	t.Log("Injection test passed!")
}

func TestNoInjectionWithoutAnnotation(t *testing.T) {
	ctx := context.Background()

	// Create pod WITHOUT injection annotation
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-no-injection-pod",
			Namespace: namespace,
			// No keeper.security/inject annotation
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "test",
					Image:   "alpine:latest",
					Command: []string{"sleep", "10"},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	createdPod, err := clientset.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test pod: %v", err)
	}
	defer clientset.CoreV1().Pods(namespace).Delete(ctx, "test-no-injection-pod", metav1.DeleteOptions{})

	// Wait briefly for webhook to process
	time.Sleep(2 * time.Second)

	// Get the pod again
	updatedPod, err := clientset.CoreV1().Pods(namespace).Get(ctx, "test-no-injection-pod", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get pod: %v", err)
	}

	// Verify NO init containers were added
	for _, ic := range updatedPod.Spec.InitContainers {
		if ic.Name == "keeper-secrets-init" {
			t.Error("Init container should NOT be injected without annotation")
		}
	}

	// Verify NO sidecar was added
	for _, c := range updatedPod.Spec.Containers {
		if c.Name == "keeper-secrets-sidecar" {
			t.Error("Sidecar should NOT be injected without annotation")
		}
	}

	t.Log("No-injection test passed!")
}

func waitForPodRunning(ctx context.Context, namespace, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		if pod.Status.Phase == corev1.PodRunning {
			return nil
		}
		if pod.Status.Phase == corev1.PodFailed {
			return fmt.Errorf("pod failed")
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout waiting for pod to be running")
}
