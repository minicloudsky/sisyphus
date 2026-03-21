package data

import (
	"context"
	"time"

	"sisyphus/server/internal/biz"
)

type userRepo struct {
	data *Data
}

// NewUserRepo .
func NewUserRepo(data *Data) biz.UserRepo {
	return &userRepo{data: data}
}

func (r *userRepo) Create(ctx context.Context, u *biz.User) (*biz.User, error) {
	user := &User{
		Username:   u.Username,
		Password:   u.Password,
		Email:      u.Email,
		RealName:   u.RealName,
		Phone:      u.Phone,
		Department: u.Department,
		Status:     int8(u.Status),
	}
	if err := r.data.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, err
	}
	return toBizUser(user), nil
}

func (r *userRepo) Update(ctx context.Context, u *biz.User) error {
	updates := map[string]interface{}{
		"email":      u.Email,
		"real_name":  u.RealName,
		"phone":      u.Phone,
		"department": u.Department,
		"status":     u.Status,
		"updated_at": time.Now(),
	}
	if u.Password != "" {
		updates["password"] = u.Password
	}
	return r.data.db.WithContext(ctx).Model(&User{}).Where("id = ?", u.ID).Updates(updates).Error
}

func (r *userRepo) Delete(ctx context.Context, id uint64) error {
	return r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&User{}).Error
}

func (r *userRepo) GetByID(ctx context.Context, id uint64) (*biz.User, error) {
	var user User
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		return nil, err
	}
	return toBizUser(&user), nil
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*biz.User, error) {
	var user User
	if err := r.data.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return toBizUser(&user), nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*biz.User, error) {
	var user User
	if err := r.data.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return toBizUser(&user), nil
}

func (r *userRepo) List(ctx context.Context, page, pageSize int) ([]*biz.User, int64, error) {
	var users []User
	var total int64

	if err := r.data.db.WithContext(ctx).Model(&User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := r.data.db.WithContext(ctx).Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.User, len(users))
	for i, u := range users {
		result[i] = toBizUser(&u)
	}
	return result, total, nil
}

// User 数据库模型
type User struct {
	ID         uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	Username   string    `gorm:"column:username;type:varchar(100);uniqueIndex;not null"`
	Password   string    `gorm:"column:password;type:varchar(255);not null"`
	Email      string    `gorm:"column:email;type:varchar(100);uniqueIndex;not null"`
	RealName   string    `gorm:"column:real_name;type:varchar(100);not null"`
	Phone      string    `gorm:"column:phone;type:varchar(20);default:''"`
	Department string    `gorm:"column:department;type:varchar(100);default:''"`
	Status     int8      `gorm:"column:status;type:tinyint;default:1;comment:1-正常,0-禁用"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (User) TableName() string {
	return "sisyphus_users"
}

func toBizUser(u *User) *biz.User {
	return &biz.User{
		ID:         u.ID,
		Username:   u.Username,
		Password:   u.Password,
		Email:      u.Email,
		RealName:   u.RealName,
		Phone:      u.Phone,
		Department: u.Department,
		Status:     int(u.Status),
		CreatedAt:  u.CreatedAt,
		UpdatedAt:  u.UpdatedAt,
	}
}
