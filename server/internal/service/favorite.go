package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	v1 "sisyphus/server/api/proto/v1"
	"sisyphus/server/internal/biz"
)

// FavoriteService 收藏夹服务
type FavoriteService struct {
	v1.UnimplementedFavoriteServiceServer

	favoriteUsecase *biz.FavoriteUsecase
	log             *log.Helper
}

// NewFavoriteService 创建收藏夹服务
func NewFavoriteService(favoriteUsecase *biz.FavoriteUsecase, logger log.Logger) *FavoriteService {
	return &FavoriteService{
		favoriteUsecase: favoriteUsecase,
		log:             log.NewHelper(logger),
	}
}

// CreateFavorite 创建收藏
func (s *FavoriteService) CreateFavorite(ctx context.Context, req *v1.CreateFavoriteRequest) (*v1.CreateFavoriteResponse, error) {
	userID := getUserIDFromContext(ctx)
	if userID == 0 {
		return nil, ErrUnauthorized
	}

	favorite := &biz.QueryFavorite{
		UserID:       userID,
		InstanceID:   req.InstanceId,
		DatabaseName: req.DatabaseName,
		Name:         req.Name,
		DBType:       req.DbType,
		Tag:          req.Tag,
		SQLText:      req.SqlText,
	}

	if err := s.favoriteUsecase.CreateFavorite(ctx, favorite); err != nil {
		if err == biz.ErrDuplicateFavorite {
			return &v1.CreateFavoriteResponse{
				Id:      0,
				Message: "收藏已存在",
			}, nil
		}
		return nil, err
	}

	return &v1.CreateFavoriteResponse{
		Id:      favorite.ID,
		Message: "success",
	}, nil
}

// UpdateFavorite 更新收藏
func (s *FavoriteService) UpdateFavorite(ctx context.Context, req *v1.UpdateFavoriteRequest) (*v1.CommonResponse, error) {
	userID := getUserIDFromContext(ctx)
	if userID == 0 {
		return nil, ErrUnauthorized
	}

	favorite := &biz.QueryFavorite{
		ID:           req.Id,
		UserID:       userID,
		InstanceID:   req.InstanceId,
		DatabaseName: req.DatabaseName,
		Name:         req.Name,
		DBType:       req.DbType,
		Tag:          req.Tag,
		SQLText:      req.SqlText,
	}

	if err := s.favoriteUsecase.UpdateFavorite(ctx, favorite); err != nil {
		return nil, err
	}

	return &v1.CommonResponse{Message: "success"}, nil
}

// DeleteFavorite 删除收藏
func (s *FavoriteService) DeleteFavorite(ctx context.Context, req *v1.DeleteFavoriteRequest) (*v1.CommonResponse, error) {
	userID := getUserIDFromContext(ctx)
	if userID == 0 {
		return nil, ErrUnauthorized
	}

	if err := s.favoriteUsecase.DeleteFavorite(ctx, req.Id, userID); err != nil {
		return nil, err
	}

	return &v1.CommonResponse{Message: "success"}, nil
}

// GetFavorite 获取收藏详情
func (s *FavoriteService) GetFavorite(ctx context.Context, req *v1.GetFavoriteRequest) (*v1.GetFavoriteResponse, error) {
	userID := getUserIDFromContext(ctx)
	if userID == 0 {
		return nil, ErrUnauthorized
	}

	favorite, err := s.favoriteUsecase.GetFavorite(ctx, req.Id, userID)
	if err != nil {
		return nil, err
	}
	if favorite == nil {
		return nil, biz.ErrFavoriteNotFound
	}

	return &v1.GetFavoriteResponse{
		Favorite: toProtoFavorite(favorite),
	}, nil
}

// ListFavorites 获取收藏列表
func (s *FavoriteService) ListFavorites(ctx context.Context, req *v1.ListFavoritesRequest) (*v1.ListFavoritesResponse, error) {
	userID := getUserIDFromContext(ctx)
	if userID == 0 {
		return nil, ErrUnauthorized
	}

	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	var instanceIDs []uint64
	if len(req.InstanceIds) > 0 {
		instanceIDs = req.InstanceIds
	}

	favorites, total, err := s.favoriteUsecase.ListFavorites(ctx, userID, instanceIDs, page, pageSize)
	if err != nil {
		return nil, err
	}

	protoFavorites := make([]*v1.Favorite, len(favorites))
	for i, f := range favorites {
		protoFavorites[i] = toProtoFavorite(f)
	}

	return &v1.ListFavoritesResponse{
		Favorites: protoFavorites,
		Page: &v1.PageResponse{
			Total:    int64(total),
			Page:     int32(page),
			PageSize: int32(pageSize),
		},
	}, nil
}

// PinFavorite 置顶收藏
func (s *FavoriteService) PinFavorite(ctx context.Context, req *v1.PinFavoriteRequest) (*v1.CommonResponse, error) {
	userID := getUserIDFromContext(ctx)
	if userID == 0 {
		return nil, ErrUnauthorized
	}

	if err := s.favoriteUsecase.PinFavorite(ctx, req.Id, userID); err != nil {
		return nil, err
	}

	return &v1.CommonResponse{Message: "success"}, nil
}

func toProtoFavorite(f *biz.QueryFavorite) *v1.Favorite {
	return &v1.Favorite{
		Id:           f.ID,
		InstanceId:   f.InstanceID,
		DatabaseName: f.DatabaseName,
		Name:         f.Name,
		DbType:       f.DBType,
		Tag:          f.Tag,
		SqlText:      f.SQLText,
		Score:        int32(f.Score),
		CreatedAt:    f.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:    f.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
