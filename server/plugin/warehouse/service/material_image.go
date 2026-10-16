package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/model/example"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/warehouse/model"
	"github.com/flipped-aurora/gin-vue-admin/server/utils/upload"
	"gorm.io/gorm"
)

const (
	MaterialImageMaxBytes       = 1_048_576
	materialImageValidationV1   = 1
)

// MaterialImageFile is the narrow file-record boundary required by material images.
type MaterialImageFile struct {
	ID  uint
	URL string
	Key string
}

// MaterialImageStorage keeps real object storage and system-file records outside aggregate rules.
type MaterialImageStorage interface {
	Store(context.Context, string, []byte) (MaterialImageFile, error)
	Delete(context.Context, MaterialImageFile) error
}

// MaterialImageRegistration persists only the warehouse authorization record.
type MaterialImageRegistration interface {
	Register(context.Context, model.MaterialImage) error
}

type gormMaterialImageRegistration struct{ db *gorm.DB }

func (r gormMaterialImageRegistration) Register(ctx context.Context, image model.MaterialImage) error {
	return r.db.WithContext(ctx).Create(&image).Error
}

// FrameworkMaterialImageStorage adapts the existing OSS abstraction and file table.
type FrameworkMaterialImageStorage struct {
	db  *gorm.DB
	oss upload.OSS
}

func NewFrameworkMaterialImageStorage(db *gorm.DB) *FrameworkMaterialImageStorage {
	return &FrameworkMaterialImageStorage{db: db, oss: upload.NewOss()}
}

func (s *FrameworkMaterialImageStorage) Store(_ context.Context, name string, contents []byte) (MaterialImageFile, error) {
	if s == nil || s.db == nil || s.oss == nil { return MaterialImageFile{}, errors.New("material image storage is unavailable") }
	var encoded bytes.Buffer
	writer := multipart.NewWriter(&encoded)
	part, err := writer.CreateFormFile("file", name)
	if err != nil { return MaterialImageFile{}, err }
	if _, err := part.Write(contents); err != nil { return MaterialImageFile{}, err }
	if err := writer.Close(); err != nil { return MaterialImageFile{}, err }
	form, err := multipart.NewReader(&encoded, writer.Boundary()).ReadForm(int64(encoded.Len()))
	if err != nil { return MaterialImageFile{}, err }
	defer form.RemoveAll()
	headers := form.File["file"]
	if len(headers) != 1 { return MaterialImageFile{}, errors.New("material image form is invalid") }
	url, key, err := s.oss.UploadFile(headers[0])
	if err != nil { return MaterialImageFile{}, err }
	record := example.ExaFileUploadAndDownload{Name: name, Url: url, Tag: strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), "."), Key: key}
	if err := s.db.Create(&record).Error; err != nil {
		_ = s.oss.DeleteFile(key)
		return MaterialImageFile{}, err
	}
	return MaterialImageFile{ID: record.ID, URL: record.Url, Key: record.Key}, nil
}

func (s *FrameworkMaterialImageStorage) Delete(_ context.Context, file MaterialImageFile) error {
	if s == nil || s.db == nil || s.oss == nil { return errors.New("material image storage is unavailable") }
	if err := s.oss.DeleteFile(file.Key); err != nil { return err }
	return s.db.Unscoped().Where("id = ?", file.ID).Delete(&example.ExaFileUploadAndDownload{}).Error
}

// MaterialImageUpload is the only image value that can be assigned to a material.
type MaterialImageUpload struct {
	FileID uint   `json:"fileId"`
	URL    string `json:"url"`
	Size   int64  `json:"size"`
	MIME   string `json:"mime"`
}

func (s *MasterService) UploadMaterialImage(ctx context.Context, actor uint, name string, contents []byte) (*MaterialImageUpload, error) {
	return registerMaterialImage(ctx, s.materialImageStorage(), gormMaterialImageRegistration{db: s.db}, actor, name, contents)
}

func registerMaterialImage(ctx context.Context, storage MaterialImageStorage, registry MaterialImageRegistration, actor uint, name string, contents []byte) (*MaterialImageUpload, error) {
	mime, err := ValidateMaterialImage(name, contents)
	if err != nil { return nil, err }
	file, err := storage.Store(ctx, name, contents)
	if err != nil { return nil, NewDomainError(ValidationFailed, "图片上传失败") }
	image := model.MaterialImage{FileID: file.ID, MIME: mime, SizeBytes: int64(len(contents)), ValidationVersion: materialImageValidationV1}
	ApplyCreateAudit(&image.BusinessModel, actor, time.Now())
	if err := registry.Register(ctx, image); err != nil {
		// Only this call's newly stored file is cleaned; existing image references remain untouched.
		_ = storage.Delete(ctx, file)
		return nil, dbError(err)
	}
	return &MaterialImageUpload{FileID: file.ID, URL: file.URL, Size: image.SizeBytes, MIME: image.MIME}, nil
}

// ValidateMaterialImage applies the exact upload boundary before object storage is called.
func ValidateMaterialImage(name string, contents []byte) (string, error) {
	if len(contents) == 0 || len(contents) > MaterialImageMaxBytes { return "", domainError(ValidationFailed, "图片大小必须小于或等于1MB", "file") }
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
	var mime string
	switch ext {
	case "jpg", "jpeg":
		if !isJPEG(contents) { return "", domainError(ValidationFailed, "图片扩展名与文件内容不匹配", "file") }
		mime = "image/jpeg"
	case "png":
		if !isPNG(contents) { return "", domainError(ValidationFailed, "图片扩展名与文件内容不匹配", "file") }
		mime = "image/png"
	case "bmp":
		if !isBMP(contents) { return "", domainError(ValidationFailed, "图片扩展名与文件内容不匹配", "file") }
		mime = "image/bmp"
	default:
		return "", domainError(ValidationFailed, "仅支持jpg、jpeg、png、bmp图片", "file")
	}
	return mime, nil
}

func isJPEG(contents []byte) bool {
	if len(contents) < 4 || contents[0] != 0xff || contents[1] != 0xd8 || contents[2] != 0xff { return false }
	_, format, err := image.DecodeConfig(bytes.NewReader(contents))
	return err == nil && format == "jpeg"
}
func isPNG(contents []byte) bool {
	if len(contents) < 24 || !bytes.Equal(contents[:8], []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}) { return false }
	_, format, err := image.DecodeConfig(bytes.NewReader(contents))
	return err == nil && format == "png"
}
func isBMP(contents []byte) bool {
	if len(contents) < 26 || contents[0] != 'B' || contents[1] != 'M' { return false }
	// A supported BMP must carry a DIB header with nonzero dimensions; signature alone is not content validation.
	dib := uint32(contents[14]) | uint32(contents[15])<<8 | uint32(contents[16])<<16 | uint32(contents[17])<<24
	width := uint32(contents[18]) | uint32(contents[19])<<8 | uint32(contents[20])<<16 | uint32(contents[21])<<24
	height := uint32(contents[22]) | uint32(contents[23])<<8 | uint32(contents[24])<<16 | uint32(contents[25])<<24
	return dib >= 12 && width > 0 && height > 0
}

func (s *MasterService) materialImageStorage() MaterialImageStorage {
	if s.images != nil { return s.images }
	return NewFrameworkMaterialImageStorage(s.db)
}

