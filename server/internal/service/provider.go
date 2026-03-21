package service

import "github.com/google/wire"

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
	NewAuthService,
	NewSQLOrderService,
	NewQueryService,
	NewBinlogService,
	NewRedisService,
	NewArchiveService,
	NewInstanceService,
	NewGhOstService,
	NewPermissionService,
	NewFavoriteService,
	NewMyBatisService,
	NewWidetableService,
	// PartitionService需要特殊处理，在wire.go中提供
)
