package data

import (
	"context"
	"storage/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type psFileRepo struct {
	data *Data
	log  *log.Helper
}

// CheckPsFileExistsByHash implements [biz.PsFileRepo].
func (p *psFileRepo) CheckPsFileExistsByHash(context.Context, string) (bool, error) {
	panic("unimplemented")
}

// CreateFsFile implements [biz.PsFileRepo].
func (p *psFileRepo) CreateFsFile(context.Context, *biz.PsFile) (*biz.PsFile, error) {
	panic("unimplemented")
}

// GetFsFileByID implements [biz.PsFileRepo].
func (p *psFileRepo) GetFsFileByID(context.Context, string) (*biz.PsFile, error) {
	panic("unimplemented")
}

// GetFsFiles implements [biz.PsFileRepo].
func (p *psFileRepo) GetFsFiles(context.Context) ([]*biz.PsFile, error) {
	panic("unimplemented")
}

// GetPsFileByHash implements [biz.PsFileRepo].
func (p *psFileRepo) GetPsFileByHash(context.Context, string) (*biz.PsFile, error) {
	panic("unimplemented")
}

func NewPsFileRepo(data *Data, logger log.Logger) biz.PsFileRepo {
	return &psFileRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}
