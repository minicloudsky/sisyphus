package service

import (
	"github.com/go-kratos/kratos/v2/log"

	"sisyphus/server/internal/biz"
)

// PartitionService 分区管理服务
type PartitionService struct {
	partitionUsecase *biz.PartitionUsecase
	log              *log.Helper
}

// NewPartitionService 创建分区管理服务
func NewPartitionService(partitionUsecase *biz.PartitionUsecase, logger log.Logger) *PartitionService {
	return &PartitionService{
		partitionUsecase: partitionUsecase,
		log:              log.NewHelper(logger),
	}
}
