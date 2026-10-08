package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/domain"
)

type FileSystem struct{ root string }

func NewFileSystem(root string) (*FileSystem, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("attachment storage root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	return &FileSystem{root: filepath.Clean(root)}, nil
}
func (s *FileSystem) Save(ctx context.Context, prefix string, reader io.Reader, max int64) (string, int64, error) {
	if max <= 0 {
		return "", 0, errors.New("attachment size limit must be positive")
	}
	dir := filepath.Join(s.root, filepath.Clean(prefix))
	if err := s.safePath(dir); err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", 0, err
	}
	tmp, err := os.CreateTemp(dir, ".upload-")
	if err != nil {
		return "", 0, err
	}
	tempName := tmp.Name()
	defer os.Remove(tempName)
	limited := io.LimitReader(reader, max+1)
	size, copyErr := io.Copy(tmp, limited)
	closeErr := tmp.Close()
	if copyErr != nil {
		return "", 0, copyErr
	}
	if closeErr != nil {
		return "", 0, closeErr
	}
	if size > max {
		return "", 0, fmt.Errorf("%w: %d bytes", domain.ErrAttachmentTooLarge, max)
	}
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}
	key := filepath.Join(prefix, uuid.New().String())
	final := filepath.Join(s.root, key)
	if err := os.Rename(tempName, final); err != nil {
		return "", 0, err
	}
	if err := os.Chmod(final, 0600); err != nil {
		return "", 0, err
	}
	return filepath.ToSlash(key), size, nil
}
func (s *FileSystem) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}
func (s *FileSystem) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
func (s *FileSystem) path(key string) (string, error) {
	path := filepath.Join(s.root, filepath.Clean(filepath.FromSlash(key)))
	if err := s.safePath(path); err != nil {
		return "", err
	}
	return path, nil
}
func (s *FileSystem) safePath(path string) error {
	relative, err := filepath.Rel(s.root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return errors.New("invalid attachment storage path")
	}
	return nil
}

var _ domain.FileStore = (*FileSystem)(nil)
