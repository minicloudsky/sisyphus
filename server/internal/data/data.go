package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"sisyphus/server/internal/conf"
)

// Data .
type Data struct {
	db  *gorm.DB
	rdb *redis.Client
}

// autoMigrateModels 在启动时执行非破坏性 schema 校验（只增不删）
func autoMigrateModels(db *gorm.DB) error {
	return db.AutoMigrate(
		&User{},
		&DatabaseInstance{},
		&Permission{},
		&Role{},
		&RolePermission{},
		&UserRole{},
		&UserInstancePermission{},
		&RoleInstancePermission{},
		&SQLOrder{},
		&SQLOrderItem{},
		&SQLReviewRecord{},
		&BinlogRollbackOrder{},
		&BinlogRollbackSQL{},
		&SQLRollback{},
		&RedisHotKeyTask{},
		&ArchiveConfig{},
		&ArchiveRecord{},
		&QueryHistory{},
		&QueryFavorite{},
		&PartitionTableInfo{},
		&PartitionOperationRecord{},
		&WidetableConfig{},
		&WidetableSyncRecord{},
		&NotificationConfig{},
	)
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	// 初始化数据库
	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 非破坏性 schema 校验（只添加缺失列/索引，不删除）
	if err := autoMigrateModels(db); err != nil {
		log.NewHelper(logger).Warnf("AutoMigrate warning: %v", err)
	}

	// 初始化Redis
	rdb := redis.NewClient(&redis.Options{
		Network:      c.Redis.Network,
		Addr:         c.Redis.Addr,
		Password:     c.Redis.Password,
		DB:           int(c.Redis.Db),
		DialTimeout: c.Redis.DialTimeout,
		ReadTimeout:  c.Redis.ReadTimeout,
		WriteTimeout: c.Redis.WriteTimeout,
	})

	// 测试Redis连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.NewHelper(logger).Warnf("redis ping failed: %v", err)
	}

	cleanup := func() {
		sqlDB.Close()
		rdb.Close()
	}

	return &Data{
		db:  db,
		rdb: rdb,
	}, cleanup, nil
}

// DB 返回数据库连接
func (d *Data) DB() *gorm.DB {
	return d.db
}

// Redis 返回Redis连接
func (d *Data) Redis() *redis.Client {
	return d.rdb
}
