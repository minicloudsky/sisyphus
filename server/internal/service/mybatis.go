package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
)

// MyBatisService MyBatis代码生成服务
type MyBatisService struct {
	v1.UnimplementedMyBatisServiceServer

	generator *biz.MyBatisGenerator
	log       *log.Helper
}

// NewMyBatisService 创建MyBatis代码生成服务
func NewMyBatisService(generator *biz.MyBatisGenerator, logger log.Logger) *MyBatisService {
	return &MyBatisService{
		generator: generator,
		log:       log.NewHelper(logger),
	}
}

// GenerateCode 生成MyBatis代码
func (s *MyBatisService) GenerateCode(ctx context.Context, req *v1.GenerateCodeRequest) (*v1.GenerateCodeResponse, error) {
	xml, err := s.generator.GenerateXML(req.TableMetaText)
	if err != nil {
		s.log.Errorf("生成MyBatis代码失败: %v", err)
		return nil, err
	}

	return &v1.GenerateCodeResponse{
		Data:     map[string]string{"mybatis": xml},
		NameList: []string{"mybatis"},
	}, nil
}
