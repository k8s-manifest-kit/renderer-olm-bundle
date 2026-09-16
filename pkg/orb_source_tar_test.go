package orb

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var tarConfigMapGVK = schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap"}

func TestUntarRejectsUnsafePaths(t *testing.T) {
	for _, name := range []string{"../outside", "/absolute"} {
		t.Run(name, func(t *testing.T) {
			g := NewWithT(t)
			archive := tarBytes(t, name, []byte("content"))
			err := untar(t.Context(), archive, t.TempDir())
			g.Expect(err).To(HaveOccurred())
		})
	}
}

func TestTarSource(t *testing.T) {
	g := NewWithT(t)
	bundleDir := filepath.Join("..", "config", "test", "bundles", "simple")
	tarPath := filepath.Join(t.TempDir(), "simple.tar")

	err := writeTar(tarPath, bundleDir)
	g.Expect(err).NotTo(HaveOccurred())

	renderer, err := New([]Source{{
		Bundle: "tar:" + tarPath,
	}})
	g.Expect(err).NotTo(HaveOccurred())

	objects, err := renderer.Process(t.Context(), nil)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(findObject(objects, tarConfigMapGVK, "simple-operator-config")).NotTo(BeNil())
}

func tarBytes(t *testing.T, name string, contents []byte) *bytes.Reader {
	t.Helper()
	var buffer bytes.Buffer
	writer := tar.NewWriter(&buffer)
	err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(contents))})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	return bytes.NewReader(buffer.Bytes())
}

func writeTar(path, sourceDir string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	writer := tar.NewWriter(file)
	defer func() { _ = writer.Close() }()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relative, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relative)
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		contents, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, contents)
		closeErr := contents.Close()
		if copyErr != nil {
			return copyErr
		}

		return closeErr
	})
}

func findObject(objects []unstructured.Unstructured, gvk schema.GroupVersionKind, name string) *unstructured.Unstructured {
	for i := range objects {
		if objects[i].GroupVersionKind() == gvk && objects[i].GetName() == name {
			return &objects[i]
		}
	}

	return nil
}
