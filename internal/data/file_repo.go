package data

import (
	"context"
	"storage/internal/biz"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

type fileRepo struct {
	data *Data
	log  *log.Helper
}

// MoveFilesToFolder implements [biz.FileRepo].
func (f *fileRepo) MoveFilesToFolder(ctx context.Context, ownerID string, folderID *uuid.UUID, ids ...uuid.UUID) error {
	panic("unimplemented")
}

// UpdateFileNameByID implements [biz.FileRepo].
func (f *fileRepo) UpdateFileNameByID(ctx context.Context, ownerID string, newName string, id uuid.UUID) error {
	panic("unimplemented")
}

// CreateFile implements [biz.FileRepo].
func (f *fileRepo) CreateFile(context.Context, *biz.File) (*biz.File, error) {
	panic("unimplemented")
}

// DeleteFiles implements [biz.FileRepo].
func (f *fileRepo) DeleteFiles(ctx context.Context, ids ...uuid.UUID) error {
	panic("unimplemented")
}

// DeleteFilesByIDsPermanentlyAuto implements [biz.FileRepo].
func (f *fileRepo) DeleteFilesByIDsPermanentlyAuto(ctx context.Context, ids ...uuid.UUID) error {
	panic("unimplemented")
}

// GetAllFilesInTrash implements [biz.FileRepo].
func (f *fileRepo) GetAllFilesInTrash(ctx context.Context, ownerID string) ([]*biz.File, error) {
	panic("unimplemented")
}

// GetFileByFileHash implements [biz.FileRepo].
func (f *fileRepo) GetFileByFileHash(ctx context.Context, folderID *uuid.UUID, ownerID string, fileHash string) (*biz.File, error) {
	panic("unimplemented")
}

// GetFiles implements [biz.FileRepo].
func (f *fileRepo) GetFiles() ([]*biz.File, error) {
	panic("unimplemented")
}

// GetFilesByIDs implements [biz.FileRepo].
func (f *fileRepo) GetFilesByIDs(ctx context.Context, ownerID string, ids ...uuid.UUID) ([]*biz.File, error) {
	panic("unimplemented")
}

// GetFilesByIDsInTrash implements [biz.FileRepo].
func (f *fileRepo) GetFilesByIDsInTrash(ctx context.Context, ownerID string, ids ...uuid.UUID) ([]*biz.File, error) {
	panic("unimplemented")
}

// GetFilesByIDsUnscoped implements [biz.FileRepo].
func (f *fileRepo) GetFilesByIDsUnscoped(ctx context.Context, ownerID string, ids ...uuid.UUID) ([]*biz.File, error) {
	panic("unimplemented")
}

// GetFilesExpiredInTrash implements [biz.FileRepo].
func (f *fileRepo) GetFilesExpiredInTrash(ctx context.Context, expiredBefore time.Time, batch int) ([]*biz.File, error) {
	panic("unimplemented")
}

// GetFoldersByFolderID implements [biz.FileRepo].
func (f *fileRepo) GetFoldersByFolderID(ctx context.Context, folderID *uuid.UUID, ownerID string) ([]*biz.File, error) {
	panic("unimplemented")
}

// RestoreFiles implements [biz.FileRepo].
func (f *fileRepo) RestoreFiles(ctx context.Context, ownerID string, ids ...uuid.UUID) error {
	panic("unimplemented")
}

// RestoreFilesToRoot implements [biz.FileRepo].
func (f *fileRepo) RestoreFilesToRoot(ctx context.Context, ownerID string, ids ...uuid.UUID) error {
	panic("unimplemented")
}

// UpdateFavoriteFilesByIDs implements [biz.FileRepo].
func (f *fileRepo) UpdateFavoriteFilesByIDs(ctx context.Context, ownerID string, favorite bool, ids ...uuid.UUID) error {
	panic("unimplemented")
}

// UpdateRecentAccessedAtByIDs implements [biz.FileRepo].
func (f *fileRepo) UpdateRecentAccessedAtByIDs(ctx context.Context, ownerID string, accessedAt time.Time, ids ...uuid.UUID) error {
	panic("unimplemented")
}

func NewFileRepo(data *Data, logger log.Logger) biz.FileRepo {
	return &fileRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}
