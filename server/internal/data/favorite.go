package data

import (
	"context"
	"time"

	"sisyphus/server/internal/biz"
	"gorm.io/gorm"
)

type QueryFavorite struct {
	ID           uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	UserID       uint64    `gorm:"column:user_id;not null;index:idx_user"`
	InstanceID   uint64    `gorm:"column:instance_id;not null;index:idx_instance"`
	DatabaseName string    `gorm:"column:database_name;type:varchar(100);not null"`
	Name         string    `gorm:"column:name;type:varchar(100);default:''"`
	DBType       string    `gorm:"column:db_type;type:varchar(20);default:''"`
	Tag          string    `gorm:"column:tag;type:varchar(256)"`
	SQLText      string    `gorm:"column:sql_text;type:text;not null"`
	Score        int       `gorm:"column:score;not null;default:0;index:idx_score"`
	CreatedAt    time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (QueryFavorite) TableName() string {
	return "sisyphus_query_favorites"
}

type favoriteRepo struct {
	data *Data
}

// NewFavoriteRepo 创建收藏夹仓储
func NewFavoriteRepo(data *Data) biz.FavoriteRepo {
	return &favoriteRepo{data: data}
}

func (r *favoriteRepo) Create(ctx context.Context, favorite *biz.QueryFavorite) error {
	f := &QueryFavorite{
		UserID:       favorite.UserID,
		InstanceID:   favorite.InstanceID,
		DatabaseName: favorite.DatabaseName,
		Name:         favorite.Name,
		DBType:       favorite.DBType,
		Tag:          favorite.Tag,
		SQLText:      favorite.SQLText,
		Score:        favorite.Score,
	}
	if err := r.data.db.WithContext(ctx).Create(f).Error; err != nil {
		return err
	}
	favorite.ID = f.ID
	favorite.CreatedAt = f.CreatedAt
	favorite.UpdatedAt = f.UpdatedAt
	return nil
}

func (r *favoriteRepo) Update(ctx context.Context, favorite *biz.QueryFavorite) error {
	updates := map[string]interface{}{
		"name":          favorite.Name,
		"instance_id":   favorite.InstanceID,
		"database_name": favorite.DatabaseName,
		"tag":           favorite.Tag,
		"sql_text":      favorite.SQLText,
		"db_type":       favorite.DBType,
		"updated_at":    time.Now(),
	}
	return r.data.db.WithContext(ctx).Model(&QueryFavorite{}).
		Where("id = ? AND user_id = ?", favorite.ID, favorite.UserID).
		Updates(updates).Error
}

func (r *favoriteRepo) Delete(ctx context.Context, id, userID uint64) error {
	return r.data.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&QueryFavorite{}).Error
}

func (r *favoriteRepo) GetByID(ctx context.Context, id, userID uint64) (*biz.QueryFavorite, error) {
	var f QueryFavorite
	if err := r.data.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&f).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return toBizFavorite(&f), nil
}

func (r *favoriteRepo) List(ctx context.Context, userID uint64, instanceIDs []uint64, page, pageSize int) ([]*biz.QueryFavorite, int64, error) {
	var favorites []QueryFavorite
	var total int64

	query := r.data.db.WithContext(ctx).Model(&QueryFavorite{}).
		Where("user_id = ?", userID)

	if len(instanceIDs) > 0 {
		query = query.Where("instance_id IN ?", instanceIDs)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("score DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&favorites).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.QueryFavorite, len(favorites))
	for i, f := range favorites {
		result[i] = toBizFavorite(&f)
	}

	return result, total, nil
}

func (r *favoriteRepo) UpdateScore(ctx context.Context, id, userID uint64, score int) error {
	return r.data.db.WithContext(ctx).Model(&QueryFavorite{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("score", score).Error
}

func toBizFavorite(f *QueryFavorite) *biz.QueryFavorite {
	return &biz.QueryFavorite{
		ID:           f.ID,
		UserID:       f.UserID,
		InstanceID:   f.InstanceID,
		DatabaseName: f.DatabaseName,
		Name:         f.Name,
		DBType:       f.DBType,
		Tag:          f.Tag,
		SQLText:      f.SQLText,
		Score:        f.Score,
		CreatedAt:    f.CreatedAt,
		UpdatedAt:    f.UpdatedAt,
	}
}
