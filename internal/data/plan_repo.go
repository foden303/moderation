package data

import (
	"context"
	"storage/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

type planRepo struct {
	data *Data
	log  *log.Helper
}

// CreatePlan implements [biz.PlanRepo].
func (p *planRepo) CreatePlan(context.Context, *biz.Plan) (*biz.Plan, error) {
	panic("unimplemented")
}

// GetPlanByID implements [biz.PlanRepo].
func (p *planRepo) GetPlanByID(context.Context, string) (*biz.Plan, error) {
	panic("unimplemented")
}

// GetPlans implements [biz.PlanRepo].
func (p *planRepo) GetPlans(context.Context) ([]*biz.Plan, error) {
	panic("unimplemented")
}

func NewPlanRepo(data *Data, logger log.Logger) biz.PlanRepo {
	return &planRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}
