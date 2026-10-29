package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/keeper-security/keeper-k8s-injector/pkg/config"
)

func podPorts(ports ...int32) *corev1.Pod {
	c := corev1.Container{Name: "app"}
	for _, p := range ports {
		c.Ports = append(c.Ports, corev1.ContainerPort{ContainerPort: p})
	}
	return &corev1.Pod{Spec: corev1.PodSpec{Containers: []corev1.Container{c}}}
}

func TestHealthPortFor(t *testing.T) {
	cases := []struct {
		name      string
		pod       *corev1.Pod
		requested int
		want      int
	}{
		{"no declared ports: default", podPorts(), 0, 8080},
		{"unrelated port: default", podPorts(3000), 0, 8080},
		{"app on 8080: next free", podPorts(8080), 0, 8081},
		{"skips every declared port", podPorts(8080, 8081, 8082), 0, 8083},
		{"annotation wins", podPorts(8080), 9090, 9090},
	}
	for _, c := range cases {
		got, err := healthPortFor(c.pod, c.requested)
		assert.NoError(t, err, c.name)
		assert.Equal(t, c.want, got, c.name)
	}
}

func TestHealthPortForSearchesPastLongRuns(t *testing.T) {
	// A long run of declared ports must not fall back to an occupied one.
	var ports []int32
	for p := int32(8080); p < 8180; p++ {
		ports = append(ports, p)
	}
	got, err := healthPortFor(podPorts(ports...), 0)
	assert.NoError(t, err)
	assert.Equal(t, 8180, got)
}

func TestHealthPortForErrorsWhenExhausted(t *testing.T) {
	var ports []int32
	for p := int32(8080); p <= 65535; p++ {
		ports = append(ports, p)
	}
	_, err := healthPortFor(podPorts(ports...), 0)
	assert.Error(t, err)
}

func TestSidecarProbesFollowHealthPort(t *testing.T) {
	m := newTestMutator(t)

	def := m.buildSidecarContainer(&config.InjectionConfig{RefreshInterval: "5m", HealthPort: 8080}, "{}")
	assert.Equal(t, 8080, def.LivenessProbe.HTTPGet.Port.IntValue())
	assert.Equal(t, 8080, def.ReadinessProbe.HTTPGet.Port.IntValue())
	assert.NotContains(t, def.Args, "--health-port=8080", "default port passes no flag, so older sidecar images keep working")

	moved := m.buildSidecarContainer(&config.InjectionConfig{RefreshInterval: "5m", HealthPort: 8081}, "{}")
	assert.Equal(t, 8081, moved.LivenessProbe.HTTPGet.Port.IntValue())
	assert.Equal(t, 8081, moved.ReadinessProbe.HTTPGet.Port.IntValue())
	assert.Contains(t, moved.Args, "--health-port=8081")
}
