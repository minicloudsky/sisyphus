package data

import "github.com/google/wire"

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewCronLeaderLock,
	NewUserRepo,
	NewSQLOrderRepo,
	NewPermissionRepo,
	NewDatabaseInstanceRepo,
	NewBinlogRollbackRepo,
	NewRedisAnalysisRepo,
	NewArchiveRepo,
	NewPartitionRepo,
	NewQueryHistoryRepo,
	NewFavoriteRepo,
	NewWidetableRepo,
	NewRollbackRepo,
	NewNotificationConfigRepo,
)
