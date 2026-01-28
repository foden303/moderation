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
	// Check exists
	// Create
	CreateFile(context.Context, *File) (*File, error)
	// Get
	GetFileByFileHash(ctx context.Context, folderID *uuid.UUID, ownerID, fileHash string) (*File, error)
	GetFilesByIDs(ctx context.Context, ownerID string, ids ...uuid.UUID) ([]*File, error)
	GetFiles() ([]*File, error)
	GetFoldersByFolderID(ctx context.Context, folderID *uuid.UUID, ownerID string) ([]*File, error)
	//Trash
	GetAllFilesInTrash(ctx context.Context, ownerID string) ([]*File, error)
	GetFilesExpiredInTrash(ctx context.Context, expiredBefore time.Time, batch int) ([]*File, error)
	GetFilesByIDsUnscoped(ctx context.Context, ownerID string, ids ...uuid.UUID) ([]*File, error)
	GetFilesByIDsInTrash(ctx context.Context, ownerID string, ids ...uuid.UUID) ([]*File, error)
	DeleteFilesByIDsPermanentlyAuto(ctx context.Context, ids ...uuid.UUID) error
	// DeleteFile permanently deletes a file.
	DeleteFiles(ctx context.Context, ids ...uuid.UUID) error
	// Restore
	RestoreFiles(ctx context.Context, ownerID string, ids ...uuid.UUID) error
	RestoreFilesToRoot(ctx context.Context, ownerID string, ids ...uuid.UUID) error
	// Move
	MoveFilesToFolder(ctx context.Context, ownerID string, folderID *uuid.UUID, ids ...uuid.UUID) error
	// Update
	UpdateFileNameByID(ctx context.Context, ownerID string, newName string, id uuid.UUID) error
	UpdateFavoriteFilesByIDs(ctx context.Context, ownerID string, favorite bool, ids ...uuid.UUID) error
	UpdateRecentAccessedAtByIDs(ctx context.Context, ownerID string, accessedAt time.Time, ids ...uuid.UUID) error
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
	return uc.repo.CreateFile(ctx, f)
}

func (uc *FileUsecase) GetFileByFileHash(ctx context.Context, folderID *uuid.UUID, ownerID, fileHash string) (*File, error) {
	log.Infof("GetFileByFileHash: %v", fileHash)
	return uc.repo.GetFileByFileHash(ctx, folderID, ownerID, fileHash)
}

func (uc *FileUsecase) GetFilesByIDs(ctx context.Context, ownerID string, ids ...uuid.UUID) ([]*File, error) {
	log.Infof("GetFilesByIDs: %v", ids)
	return uc.repo.GetFilesByIDs(ctx, ownerID, ids...)
}

func (uc *FileUsecase) GetFoldersByFolderID(ctx context.Context, folderID *uuid.UUID, ownerID string) ([]*File, error) {
	log.Infof("GetFoldersByFolderID: %v", folderID)
	return uc.repo.GetFoldersByFolderID(ctx, folderID, ownerID)
}

func (uc *FileUsecase) GetAllFilesInTrash(ctx context.Context, ownerID string) ([]*File, error) {
	log.Infof("GetAllFilesInTrash for ownerID: %v", ownerID)
	return uc.repo.GetAllFilesInTrash(ctx, ownerID)
}

func (uc *FileUsecase) DeleteFiles(ctx context.Context, ids ...uuid.UUID) error {
	log.Infof("DeleteFiles: %v", ids)
	return uc.repo.DeleteFiles(ctx, ids...)
}

func (uc *FileUsecase) RestoreFiles(ctx context.Context, ownerID string, ids ...uuid.UUID) error {
	log.Infof("RestoreFiles: %v", ids)
	return uc.repo.RestoreFiles(ctx, ownerID, ids...)
}

func (uc *FileUsecase) RestoreFilesToRoot(ctx context.Context, ownerID string, ids ...uuid.UUID) error {
	log.Infof("RestoreFilesToRoot: %v", ids)
	return uc.repo.RestoreFilesToRoot(ctx, ownerID, ids...)
}

func (uc *FileUsecase) UpdateFavoriteFilesByIDs(ctx context.Context, ownerID string, favorite bool, ids ...uuid.UUID) error {
	log.Infof("UpdateFavoriteFilesByIDs: %v", ids)
	return uc.repo.UpdateFavoriteFilesByIDs(ctx, ownerID, favorite, ids...)
}

func (uc *FileUsecase) UpdateRecentAccessedAtByIDs(ctx context.Context, ownerID string, accessedAt time.Time, ids ...uuid.UUID) error {
	log.Infof("UpdateRecentAccessedAtByIDs: %v", ids)
	return uc.repo.UpdateRecentAccessedAtByIDs(ctx, ownerID, accessedAt, ids...)
}

func (uc *FileUsecase) GetFiles() ([]*File, error) {
	log.Info("GetFiles")
	return uc.repo.GetFiles()
}

func (uc *FileUsecase) GetFilesExpiredInTrash(ctx context.Context, expiredBefore time.Time, batch int) ([]*File, error) {
	log.Infof("GetFilesExpiredInTrash before: %v", expiredBefore)
	return uc.repo.GetFilesExpiredInTrash(ctx, expiredBefore, batch)
}

func (uc *FileUsecase) GetFilesByIDsUnscoped(ctx context.Context, ownerID string, ids ...uuid.UUID) ([]*File, error) {
	log.Infof("GetFilesByIDsUnscoped: %v", ids)
	return uc.repo.GetFilesByIDsUnscoped(ctx, ownerID, ids...)
}

func (uc *FileUsecase) GetFilesByIDsInTrash(ctx context.Context, ownerID string, ids ...uuid.UUID) ([]*File, error) {
	log.Infof("GetFilesByIDsInTrash: %v", ids)
	return uc.repo.GetFilesByIDsInTrash(ctx, ownerID, ids...)
}

func (uc *FileUsecase) DeleteFilesByIDsPermanentlyAuto(ctx context.Context, ids ...uuid.UUID) error {
	log.Infof("DeleteFilesByIDsPermanentlyAuto: %v", ids)
	return uc.repo.DeleteFilesByIDsPermanentlyAuto(ctx, ids...)
}

func (uc *FileUsecase) GetFilesInTrashExpiredBefore(ctx context.Context, expiredBefore time.Time, batch int) ([]*File, error) {
	log.Infof("GetFilesInTrashExpiredBefore before: %v", expiredBefore)
	return uc.repo.GetFilesExpiredInTrash(ctx, expiredBefore, batch)
}
