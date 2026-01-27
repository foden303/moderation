package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// File is a File model.
type File struct {
	CreatedAt           time.Time
	UpdatedAt           time.Time
	ID                  uuid.UUID
	FolderID            *uuid.UUID
	OwnerID             string
	Name                string
	Type                string
	Space               string
	FileHash            *string
	FileSize            *int64
	FileType            *string
	FileExt             *string
	FileMimeType        *string
	FileVideoResolution *string
	RecentAccessedAt    *time.Time
	Platform            int64
	Shared              bool
	Favorite            bool
	DeletedAt           *time.Time
	DeletedBy           *string
	UniqueHash          *string
	DelSignature        *string
}

// FileRepo is a File repo.
type FileRepo interface {
	Save(context.Context, *File) (*File, error)
	Update(context.Context, *File) (*File, error)
	FindByID(context.Context, int64) (*File, error)
	ListByHello(context.Context, string) ([]*File, error)
	ListAll(context.Context) ([]*File, error)
}

// FileUsecase is a File usecase.
type FileUsecase struct {
	repo FileRepo
}

// NewFileUsecase new a File usecase.
func NewFileUsecase(repo FileRepo) *FileUsecase {
	return &FileUsecase{repo: repo}
}

// CreateFile creates a File, and returns the new File.
func (uc *FileUsecase) CreateFile(ctx context.Context, f *File) (*File, error) {
	log.Infof("CreateFile: %v", f.Name)
	return uc.repo.Save(ctx, f)
}
