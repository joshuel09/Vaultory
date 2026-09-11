package imagestore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Filesystem stores objects under a root directory. Development only.
type Filesystem struct {
	root string
}

func NewFilesystem(root string) (*Filesystem, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve image store root: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("create image store root: %w", err)
	}
	return &Filesystem{root: abs}, nil
}

// path refuses any key that would escape the root. Keys are generated internally, not supplied by
// collectors, but a store that can be talked out of its own directory is a store that will be.
func (f *Filesystem) path(key string) (string, error) {
	if key == "" || strings.Contains(key, "..") || filepath.IsAbs(key) {
		return "", fmt.Errorf("invalid image key %q", key)
	}
	full := filepath.Join(f.root, filepath.FromSlash(key))
	if !strings.HasPrefix(full, f.root+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid image key %q", key)
	}
	return full, nil
}

func (f *Filesystem) Put(_ context.Context, key string, data []byte) error {
	full, err := f.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return fmt.Errorf("create image directory: %w", err)
	}
	// Write to a temporary file and rename, so a crash mid-write cannot leave a truncated image
	// that later reads would serve as if it were whole.
	tmp, err := os.CreateTemp(filepath.Dir(full), ".partial-*")
	if err != nil {
		return fmt.Errorf("create temporary image file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write image: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close image: %w", err)
	}
	if err := os.Rename(tmpName, full); err != nil {
		return fmt.Errorf("commit image: %w", err)
	}
	return nil
}

func (f *Filesystem) Open(_ context.Context, key string) (io.ReadCloser, error) {
	full, err := f.path(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(full)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open image: %w", err)
	}
	return file, nil
}

func (f *Filesystem) Delete(_ context.Context, key string) error {
	full, err := f.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(full); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("delete image: %w", err)
	}
	return nil
}
