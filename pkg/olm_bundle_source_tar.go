package olmbundle

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	registryv1 "github.com/joelanford/library-olm/bundle/registry/v1"
	"go.podman.io/image/v5/pkg/compression"
)

var (
	errInvalidTarPath      = errors.New("invalid tar path")
	errTarLinksUnsupported = errors.New("tar links are not supported")
	errTarFileTooLarge     = errors.New("tar file exceeds size limit")
)

const (
	directoryMode  = 0750
	fileMode       = 0600
	maxTarFileSize = 100 * 1024 * 1024
)

type tarBundleSource struct {
	path string
}

func (s *tarBundleSource) Read(ctx context.Context) (registryv1.Bundle, error) {
	return readFromTar(ctx, s.path)
}

func readFromTar(ctx context.Context, path string) (registryv1.Bundle, error) {
	tmpDir, err := os.MkdirTemp("", "olm-bundle-tar-")
	if err != nil {
		return registryv1.Bundle{}, fmt.Errorf("creating temporary tar directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	f, err := os.Open(path) //nolint:gosec // path is an explicit renderer source reference.
	if err != nil {
		return registryv1.Bundle{}, fmt.Errorf("opening tar file: %w", err)
	}
	defer func() { _ = f.Close() }()

	decompressed, _, err := compression.AutoDecompress(f)
	if err != nil {
		return registryv1.Bundle{}, fmt.Errorf("decompressing archive: %w", err)
	}
	defer func() { _ = decompressed.Close() }()

	if err := untar(ctx, decompressed, tmpDir); err != nil {
		return registryv1.Bundle{}, fmt.Errorf("extracting tar: %w", err)
	}

	bundle, err := registryv1.FromFS(os.DirFS(tmpDir))
	if err != nil {
		return registryv1.Bundle{}, fmt.Errorf("parsing bundle from tar: %w", err)
	}

	return bundle, nil
}

func untar(ctx context.Context, reader io.Reader, dest string) error {
	tarReader := tar.NewReader(reader)
	cleanDest := filepath.Clean(dest)

	for {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("checking extraction context: %w", err)
		}

		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading tar header: %w", err)
		}

		cleanName, err := safeTarPath(header.Name)
		if err != nil {
			return err
		}
		if cleanName == "." {
			continue
		}

		target := filepath.Join(cleanDest, cleanName)
		if err := extractTarEntry(tarReader, target, header); err != nil {
			return err
		}
	}
}

func safeTarPath(name string) (string, error) {
	cleanName := filepath.Clean(name)
	if filepath.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: %s", errInvalidTarPath, name)
	}

	return cleanName, nil
}

func extractTarEntry(reader io.Reader, target string, header *tar.Header) error {
	switch header.Typeflag {
	case tar.TypeDir:
		if err := os.MkdirAll(target, directoryMode); err != nil {
			return fmt.Errorf("creating directory %q: %w", header.Name, err)
		}
	case tar.TypeReg:
		return extractTarFile(reader, target, header)
	case tar.TypeSymlink, tar.TypeLink:
		return fmt.Errorf("%w: %s", errTarLinksUnsupported, header.Name)
	default:
		return nil
	}

	return nil
}

func extractTarFile(reader io.Reader, target string, header *tar.Header) error {
	if header.Size < 0 || header.Size > maxTarFileSize {
		return fmt.Errorf("%w: %q exceeds %d-byte limit", errTarFileTooLarge, header.Name, maxTarFileSize)
	}
	if err := os.MkdirAll(filepath.Dir(target), directoryMode); err != nil {
		return fmt.Errorf("creating parent directory for %q: %w", header.Name, err)
	}

	outFile, err := os.OpenFile( //nolint:gosec // target is confined below the temporary destination.
		target,
		os.O_WRONLY|os.O_CREATE|os.O_TRUNC,
		fileMode,
	)
	if err != nil {
		return fmt.Errorf("creating tar file %q: %w", header.Name, err)
	}

	_, copyErr := io.CopyN(outFile, reader, header.Size)
	closeErr := outFile.Close()
	if copyErr != nil {
		return fmt.Errorf("copying tar file %q: %w", header.Name, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing tar file %q: %w", header.Name, closeErr)
	}

	return nil
}
