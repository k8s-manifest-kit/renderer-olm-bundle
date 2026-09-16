package olmbundle

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	imageTypes "go.podman.io/image/v5/types"
)

func TestBuildSystemContextCredentials(t *testing.T) {
	ambient, err := AmbientCredentials(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		credentials *Credentials
		want        *imageTypes.DockerAuthConfig
	}{
		{
			name: "nil credentials are anonymous",
			want: &imageTypes.DockerAuthConfig{},
		},
		{
			name:        "explicit credentials",
			credentials: &Credentials{Username: "user", Password: "pass"},
			want:        &imageTypes.DockerAuthConfig{Username: "user", Password: "pass"},
		},
		{
			name:        "ambient credentials are explicit",
			credentials: ambient,
		},
		{
			name:        "empty credentials are anonymous",
			credentials: &Credentials{},
			want:        &imageTypes.DockerAuthConfig{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			g := NewWithT(t)
			systemContext := buildSystemContext(sourceOptions{credentials: test.credentials})
			g.Expect(systemContext.DockerAuthConfig).To(Equal(test.want))
		})
	}
}

func TestCredentialsReturnsNilCallbackResult(t *testing.T) {
	g := NewWithT(t)
	holder := &sourceHolder{
		Source: Source{
			Bundle: "docker://quay.io/example/bundle:v1",
			Credentials: func(context.Context) (*Credentials, error) {
				return nil, nil //nolint:nilnil // nil is the anonymous credential result.
			},
		},
		ref: transportRef{transport: transportDocker, ref: "quay.io/example/bundle:v1"},
	}

	credentials, err := (&Renderer{}).credentials(t.Context(), holder)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(credentials).To(BeNil())
}
