package biz

import (
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(
	NewUserUsecase,
	NewPermissionUsecase,
	NewSQLTypeChecker,
	NewSQLOrderUsecase,
	NewQueryExecutor,
	NewBinlogRollbackUsecase,
	NewRedisAnalysisUsecase,
	NewDatabaseInstanceUsecase,
	NewSQLExecutorSelector,
	NewQueryHistoryUsecase,
	NewWidetableUsecase,
	// ArchiveUsecase、PartitionUsecase、GhOstManager、SQLExecutor、InceptionConfig、NotificationService、AuthUsecase、RollbackUsecase和FavoriteUsecase需要特殊处理，在wire.go中提供
)
