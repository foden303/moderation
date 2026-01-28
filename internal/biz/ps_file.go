package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// PsFile is a PsFile model.
type PsFile struct {
	CreatedAt             time.Time
	UpdatedAt             time.Time
	ID                    string
	Name                  string
	DisplayName           string
	UserIdentifier        string
	EffectiveStorageQuota int64
	StoragePhotosUsed     int64
	StorageVideoUsed      int64
	StorageDocumentUsed   int64
	StorageAudioUsed      int64
	StorageCompressUsed   int64
	StorageOtherUsed      int64
	StorageTotalUsed      int64
}

// PsFileRepo is a PsFile repo.
type PsFileRepo interface {
	CreateFsFile(context.Context, *PsFile) (*PsFile, error)
	GetFsFileByID(context.Context, string) (*PsFile, error)
	GetFsFiles(context.Context) ([]*PsFile, error)
	GetPsFileByHash(context.Context, string) (*PsFile, error)
	CheckPsFileExistsByHash(context.Context, string) (bool, error)
}

// PsFileUsecase is a PsFile usecase.
type PsFileUsecase struct {
	repo PsFileRepo
}

// NewFsFileUsecase new a PsFile usecase.
func NewFsFileUsecase(repo PsFileRepo) *PsFileUsecase {
	return &PsFileUsecase{repo: repo}
}

// CreateFsFile creates a PsFile, and returns the new PsFile.
func (uc *PsFileUsecase) CreateFsFile(ctx context.Context, p *PsFile) (*PsFile, error) {
	log.Infof("CreateFsFile: %v", p.Name)
	return uc.repo.CreateFsFile(ctx, p)
}

// GetFsFileByID retrieves a PsFile by ID.
func (uc *PsFileUsecase) GetFsFileByID(ctx context.Context, id string) (*PsFile, error) {
	log.Infof("GetFsFileByID: %v", id)
	return uc.repo.GetFsFileByID(ctx, id)
}

// GetFsFiles retrieves all PsFiles.
func (uc *PsFileUsecase) GetFsFiles(ctx context.Context) ([]*PsFile, error) {
	log.Info("GetFsFiles")
	return uc.repo.GetFsFiles(ctx)
}

// GetPsFileByHash retrieves a PsFile by hash.
func (uc *PsFileUsecase) GetPsFileByHash(ctx context.Context, hash string) (*PsFile, error) {
	log.Infof("GetPsFileByHash: %v", hash)
	return uc.repo.GetPsFileByHash(ctx, hash)
}

// CheckPsFileExistsByHash checks if a PsFile exists by hash.
func (uc *PsFileUsecase) CheckPsFileExistsByHash(ctx context.Context, hash string) (bool, error) {
	log.Infof("CheckPsFileExistsByHash: %v", hash)
	return uc.repo.CheckPsFileExistsByHash(ctx, hash)
}
