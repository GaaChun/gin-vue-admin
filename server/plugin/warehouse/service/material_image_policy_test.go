package service

import (
	"context"
	"errors"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
)

type memoryImageStorage struct {
	stored  MaterialImageFile
	deletes []MaterialImageFile
}

func (s *memoryImageStorage) Store(_ context.Context, _ string, _ []byte) (MaterialImageFile, error) { return s.stored, nil }
func (s *memoryImageStorage) Delete(_ context.Context, file MaterialImageFile) error {
	s.deletes = append(s.deletes, file)
	return nil
}

type failingImageRegistration struct{}
func (failingImageRegistration) Register(context.Context, model.MaterialImage) error { return errors.New("registration failed") }

func TestMaterialImageRegistrationFailureCleansOnlyNewUpload(t *testing.T) {
	storage := &memoryImageStorage{stored: MaterialImageFile{ID: 9, URL: "new", Key: "new-key"}}
	_, err := registerMaterialImage(context.Background(), storage, failingImageRegistration{}, 1, "image.jpg", encodedImage(t, "jpeg"))
	if err == nil { t.Fatal("registration failure accepted") }
	if len(storage.deletes) != 1 || storage.deletes[0].ID != 9 { t.Fatalf("cleanup=%#v, want only new upload", storage.deletes) }
}
