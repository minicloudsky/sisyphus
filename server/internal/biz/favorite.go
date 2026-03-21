package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// QueryFavorite SQL收藏夹
type QueryFavorite struct {
	ID           uint64
	UserID       uint64
	InstanceID   uint64
	DatabaseName string
	Name         string
	DBType       string
	Tag          string
	SQLText      string
	Score        int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// FavoriteRepo 收藏夹仓储接口
type FavoriteRepo interface {
	Create(ctx context.Context, favorite *QueryFavorite) error
	Update(ctx context.Context, favorite *QueryFavorite) error
	Delete(ctx context.Context, id, userID uint64) error
	GetByID(ctx context.Context, id, userID uint64) (*QueryFavorite, error)
	List(ctx context.Context, userID uint64, instanceIDs []uint64, page, pageSize int) ([]*QueryFavorite, int64, error)
	UpdateScore(ctx context.Context, id, userID uint64, score int) error
}

// FavoriteUsecase 收藏夹业务逻辑
type FavoriteUsecase struct {
	repo FavoriteRepo
	log  *log.Helper
}

// NewFavoriteUsecase 创建收藏夹业务逻辑
func NewFavoriteUsecase(repo FavoriteRepo, logger log.Logger) *FavoriteUsecase {
	return &FavoriteUsecase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// CreateFavorite 创建收藏
func (uc *FavoriteUsecase) CreateFavorite(ctx context.Context, favorite *QueryFavorite) error {
	// 检查是否已存在相同的收藏（用户、实例、数据库、SQL都相同）
	favorites, _, err := uc.repo.List(ctx, favorite.UserID, []uint64{favorite.InstanceID}, 1, 1000)
	if err != nil {
		return err
	}
	for _, f := range favorites {
		if f.InstanceID == favorite.InstanceID &&
			f.DatabaseName == favorite.DatabaseName &&
			f.SQLText == favorite.SQLText {
			return ErrDuplicateFavorite
		}
	}

	// 获取当前用户的最大score，新收藏的score = max_score + 1
	favorites, _, err = uc.repo.List(ctx, favorite.UserID, nil, 1, 1)
	if err != nil {
		return err
	}
	maxScore := 0
	if len(favorites) > 0 {
		maxScore = favorites[0].Score
	}
	favorite.Score = maxScore + 1

	// 如果没有tag，自动生成tag
	if favorite.Tag == "" {
		favorite.Tag = favorite.DatabaseName
	}

	return uc.repo.Create(ctx, favorite)
}

// UpdateFavorite 更新收藏
func (uc *FavoriteUsecase) UpdateFavorite(ctx context.Context, favorite *QueryFavorite) error {
	existing, err := uc.repo.GetByID(ctx, favorite.ID, favorite.UserID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrFavoriteNotFound
	}
	// HTTP body 常为部分字段，避免零值覆盖已有实例/库名等
	if favorite.Name != "" {
		existing.Name = favorite.Name
	}
	if favorite.SQLText != "" {
		existing.SQLText = favorite.SQLText
	}
	if favorite.InstanceID != 0 {
		existing.InstanceID = favorite.InstanceID
	}
	if favorite.DatabaseName != "" {
		existing.DatabaseName = favorite.DatabaseName
	}
	if favorite.Tag != "" {
		existing.Tag = favorite.Tag
	}
	if favorite.DBType != "" {
		existing.DBType = favorite.DBType
	}
	return uc.repo.Update(ctx, existing)
}

// DeleteFavorite 删除收藏
func (uc *FavoriteUsecase) DeleteFavorite(ctx context.Context, id, userID uint64) error {
	return uc.repo.Delete(ctx, id, userID)
}

// GetFavorite 获取收藏详情
func (uc *FavoriteUsecase) GetFavorite(ctx context.Context, id, userID uint64) (*QueryFavorite, error) {
	return uc.repo.GetByID(ctx, id, userID)
}

// ListFavorites 获取收藏列表
func (uc *FavoriteUsecase) ListFavorites(ctx context.Context, userID uint64, instanceIDs []uint64, page, pageSize int) ([]*QueryFavorite, int64, error) {
	return uc.repo.List(ctx, userID, instanceIDs, page, pageSize)
}

// PinFavorite 置顶收藏（将score设置为最大值+1）
func (uc *FavoriteUsecase) PinFavorite(ctx context.Context, id, userID uint64) error {
	// 获取当前用户的最大score
	favorites, _, err := uc.repo.List(ctx, userID, nil, 1, 1)
	if err != nil {
		return err
	}
	maxScore := 0
	if len(favorites) > 0 {
		maxScore = favorites[0].Score
	}
	return uc.repo.UpdateScore(ctx, id, userID, maxScore+1)
}
