package data

import (
	"context"
	"time"

	"sisyphus/server/internal/biz"
)

type databaseInstanceRepo struct {
	data *Data
}

// NewDatabaseInstanceRepo 创建数据库实例仓储
func NewDatabaseInstanceRepo(data *Data) biz.DatabaseInstanceRepo {
	return &databaseInstanceRepo{data: data}
}

func (r *databaseInstanceRepo) Create(ctx context.Context, instance *biz.DatabaseInstance) (*biz.DatabaseInstance, error) {
	inst := &DatabaseInstance{
		Name:        instance.Name,
		DBType:      instance.DBType,
		Host:        instance.Host,
		Port:        instance.Port,
		Username:    instance.Username,
		Password:    instance.Password,
		Env:         instance.Env,
		Region:      instance.Region,
		Status:      int8(instance.Status),
		Owner:       instance.Owner,
		Description: instance.Description,
	}
	if err := r.data.db.WithContext(ctx).Create(inst).Error; err != nil {
		return nil, err
	}
	return toBizDatabaseInstance(inst), nil
}

func (r *databaseInstanceRepo) Update(ctx context.Context, instance *biz.DatabaseInstance) error {
	updates := map[string]interface{}{
		"name":             instance.Name,
		"db_type":          instance.DBType,
		"host":             instance.Host,
		"port":             instance.Port,
		"username":         instance.Username,
		"env":         instance.Env,
		"region":      instance.Region,
		"status":      instance.Status,
		"owner":       instance.Owner,
		"description":      instance.Description,
		"updated_at":       time.Now(),
	}
	if instance.Password != "" {
		updates["password"] = instance.Password
	}
	return r.data.db.WithContext(ctx).Model(&DatabaseInstance{}).Where("id = ?", instance.ID).Updates(updates).Error
}

func (r *databaseInstanceRepo) Delete(ctx context.Context, id uint64) error {
	return r.data.db.WithContext(ctx).Where("id = ?", id).Delete(&DatabaseInstance{}).Error
}

func (r *databaseInstanceRepo) GetByID(ctx context.Context, id uint64) (*biz.DatabaseInstance, error) {
	var instance DatabaseInstance
	if err := r.data.db.WithContext(ctx).Where("id = ?", id).First(&instance).Error; err != nil {
		return nil, err
	}
	return toBizDatabaseInstance(&instance), nil
}

func (r *databaseInstanceRepo) List(ctx context.Context, page, pageSize int, dbType *string, env *string, restrictInstanceIDs bool, allowedInstanceIDs []uint64) ([]*biz.DatabaseInstance, int64, error) {
	var instances []DatabaseInstance
	var total int64

	query := r.data.db.WithContext(ctx).Model(&DatabaseInstance{})
	if dbType != nil {
		query = query.Where("db_type = ?", *dbType)
	}
	if env != nil {
		query = query.Where("env = ?", *env)
	}
	if restrictInstanceIDs {
		if len(allowedInstanceIDs) == 0 {
			return nil, 0, nil
		}
		query = query.Where("id IN ?", allowedInstanceIDs)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&instances).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*biz.DatabaseInstance, len(instances))
	for i, inst := range instances {
		result[i] = toBizDatabaseInstance(&inst)
	}
	return result, total, nil
}

// DatabaseInstance 数据库模型
type DatabaseInstance struct {
	ID          uint64    `gorm:"column:id;primaryKey;autoIncrement"`
	Name        string    `gorm:"column:name;type:varchar(100);not null"`
	DBType      string    `gorm:"column:db_type;type:varchar(32);not null;index"`
	Host        string    `gorm:"column:host;type:varchar(512);not null"`
	Port        int       `gorm:"column:port;not null"`
	Username    string    `gorm:"column:username;type:varchar(150);not null"`
	Password    string    `gorm:"column:password;type:varchar(255);not null"`
	Env         string    `gorm:"column:env;type:varchar(50);not null;index"`
	Region      string    `gorm:"column:region;type:varchar(100);default:''"`
	Status      int8      `gorm:"column:status;type:tinyint;default:1;index"`
	Owner       string    `gorm:"column:owner;type:varchar(100);default:''"`
	Description string    `gorm:"column:description;type:varchar(500);default:''"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (DatabaseInstance) TableName() string {
	return "sisyphus_database_instances"
}

func toBizDatabaseInstance(inst *DatabaseInstance) *biz.DatabaseInstance {
	return &biz.DatabaseInstance{
		ID:          inst.ID,
		Name:        inst.Name,
		DBType:      inst.DBType,
		Host:        inst.Host,
		Port:        inst.Port,
		Username:    inst.Username,
		Password:    inst.Password,
		Env:         inst.Env,
		Region:      inst.Region,
		Status:      int(inst.Status),
		Owner:       inst.Owner,
		Description: inst.Description,
		CreatedAt:   inst.CreatedAt,
		UpdatedAt:   inst.UpdatedAt,
	}
}
