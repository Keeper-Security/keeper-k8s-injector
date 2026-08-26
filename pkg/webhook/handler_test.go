package webhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func newTestMutator(t *testing.T) *PodMutator {
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	m := NewPodMutator(c, zap.NewNop(), DefaultWebhookConfig())
	require.NoError(t, m.InjectDecoder(admission.NewDecoder(scheme)))
	return m
}

// TestHandle_RejectsNamespaceMismatch is the regression test for the core fix: the pod
// embedded in the request body must declare the same namespace as the trusted admission
// envelope. A forged request naming one namespace in the envelope and another in the body
// must be rejected before any Secret lookup happens.
func TestHandle_RejectsNamespaceMismatch(t *testing.T) {
	m := newTestMutator(t)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "poc",
			Namespace: "victim-ns",
			Annotations: map[string]string{
				"keeper.security/inject": "true",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "busybox"}}},
	}
	req := buildAdmissionRequest(t, "attacker-ns", pod)

	resp := m.Handle(context.Background(), req)

	assert.False(t, resp.Allowed)
	require.NotNil(t, resp.Result)
	assert.Contains(t, resp.Result.Message, "does not match request namespace")
}

// TestHandle_AllowsMatchingNamespace confirms the guard does not reject ordinary requests
// where the envelope and the pod body agree, which is what every legitimate call through the
// API server looks like.
func TestHandle_AllowsMatchingNamespace(t *testing.T) {
	m := newTestMutator(t)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "team-a",
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "busybox"}}},
	}
	req := buildAdmissionRequest(t, "team-a", pod)

	resp := m.Handle(context.Background(), req)

	assert.True(t, resp.Allowed)
}

// TestHandle_FillsEmptyPodNamespaceFromRequest covers a raw manifest that omits
// metadata.namespace and relies on the request URL/context to supply it — a legitimate
// pattern, not just the forged-request case. It must not be rejected, and the namespace
// used downstream must come from the trusted envelope.
func TestHandle_FillsEmptyPodNamespaceFromRequest(t *testing.T) {
	m := newTestMutator(t)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "busybox"}}},
	}
	req := buildAdmissionRequest(t, "team-a", pod)

	resp := m.Handle(context.Background(), req)

	assert.True(t, resp.Allowed)
}

// TestHandle_ExcludedNamespaceStillWorksWhenNamespacesMatch confirms the mismatch guard
// doesn't interfere with the ordinary ExcludedNamespaces behavior for legitimate requests.
func TestHandle_ExcludedNamespaceStillWorksWhenNamespacesMatch(t *testing.T) {
	m := newTestMutator(t)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "kube-system"},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "busybox"}}},
	}
	req := buildAdmissionRequest(t, "kube-system", pod)

	resp := m.Handle(context.Background(), req)

	assert.True(t, resp.Allowed)
	require.NotNil(t, resp.Result)
	assert.Contains(t, resp.Result.Message, "excluded")
}

func buildAdmissionRequest(t *testing.T, reqNamespace string, pod *corev1.Pod) admission.Request {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	raw, err := runtime.Encode(clientgoscheme.Codecs.LegacyCodec(corev1.SchemeGroupVersion), pod)
	require.NoError(t, err)
	return admission.Request{AdmissionRequest: admissionv1.AdmissionRequest{
		UID:       "test",
		Namespace: reqNamespace,
		Operation: admissionv1.Create,
		Object:    runtime.RawExtension{Raw: raw},
	}}
}

// TestHandle_MutationFailureReturnsGenericError is the regression test for the
// error-sanitization fix: a failed mutation (e.g. the auth Secret doesn't exist, or exists
// but holds no KSM config) must not leak that detail to the caller. The caller sees a fixed
// message regardless of which of those was the real cause.
func TestHandle_MutationFailureReturnsGenericError(t *testing.T) {
	m := newTestMutator(t)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "app",
			Namespace: "team-a",
			Annotations: map[string]string{
				"keeper.security/inject":          "true",
				"keeper.security/ksm-config":      "no-such-secret",
				"keeper.security/inject-env-vars": "true",
				"keeper.security/secret":          "demo",
			},
		},
		Spec: corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "busybox"}}},
	}
	req := buildAdmissionRequest(t, "team-a", pod)

	resp := m.Handle(context.Background(), req)

	assert.False(t, resp.Allowed)
	require.NotNil(t, resp.Result)
	assert.Equal(t, errPodMutationFailed.Error(), resp.Result.Message)
	assert.NotContains(t, resp.Result.Message, "no-such-secret")
}
