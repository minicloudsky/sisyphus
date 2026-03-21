import { PageContainer } from '@ant-design/pro-components';
import { Button, Table, Tag, Space, message, Tooltip, Alert, Typography } from 'antd';
import { SyncOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { listPermissions, createPermission } from '@/services/api';
import { useRequest } from 'ahooks';
import { useState, useMemo } from 'react';
import { useIntl } from '@/locales';
import {
  SYSTEM_PERMISSIONS,
  PERM_MODULE_ORDER,
  MODULE_COLOR,
  type PermModuleKey,
} from '../permissionsCatalog';

const { Text } = Typography;

const ACTION_COLOR: Record<string, string> = {
  view: 'blue',
  create: 'green',
  update: 'orange',
  delete: 'red',
  execute: 'purple',
  approve: 'gold',
  export: 'cyan',
  manage: 'geekblue',
};

export { SYSTEM_PERMISSIONS } from '../permissionsCatalog';

export default () => {
  const intl = useIntl();
  const [syncing, setSyncing] = useState(false);

  const actionLabel = useMemo(
    () => ({
      view: intl.formatMessage({ id: 'perm.action.view' }),
      create: intl.formatMessage({ id: 'perm.action.create' }),
      update: intl.formatMessage({ id: 'perm.action.update' }),
      delete: intl.formatMessage({ id: 'perm.action.delete' }),
      execute: intl.formatMessage({ id: 'perm.action.execute' }),
      approve: intl.formatMessage({ id: 'perm.action.approve' }),
      export: intl.formatMessage({ id: 'perm.action.export' }),
      manage: intl.formatMessage({ id: 'perm.action.manage' }),
    }),
    [intl]
  );

  const { data: existingPermsData, refresh } = useRequest(async () => {
    const pageSize = 200;
    let page = 1;
    const all: any[] = [];
    while (true) {
      const res = await listPermissions({ page, pageSize } as any);
      if (res?.code !== 200) break;
      const chunk = res.list || [];
      all.push(...chunk);
      if (chunk.length < pageSize) break;
      page += 1;
    }
    return { list: all, code: 200 as const };
  });
  const existingCodes = new Set(
    (existingPermsData?.list || []).map((p: any) => p.code)
  );
  const missingCount = SYSTEM_PERMISSIONS.filter((p) => !existingCodes.has(p.code)).length;

  const handleSync = async () => {
    setSyncing(true);
    const missing = SYSTEM_PERMISSIONS.filter((p) => !existingCodes.has(p.code));
    let ok = 0;
    for (const p of missing) {
      try {
        const res = await createPermission({
          name: intl.formatMessage({ id: p.nameKey }),
          code: p.code,
          resource: p.resource,
          action: p.action,
          description: intl.formatMessage({ id: p.descKey }),
        } as any);
        if (res.code === 200) ok++;
      } catch (_) {
        /* continue */
      }
    }
    message.success(intl.formatMessage({ id: 'perm.sync.done' }, { count: ok }));
    refresh();
    setSyncing(false);
  };

  const columns = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'perm.col.permName' }),
        dataIndex: 'nameKey',
        width: 130,
        render: (_: string, record: (typeof SYSTEM_PERMISSIONS)[0]) => (
          <Space>
            <span style={{ fontWeight: 500 }}>{intl.formatMessage({ id: record.nameKey })}</span>
            {existingCodes.has(record.code) ? (
              <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 13 }} />
            ) : (
              <Tooltip title={intl.formatMessage({ id: 'perm.notSyncedTip' })}>
                <span style={{ color: '#faad14', fontSize: 12 }}>●</span>
              </Tooltip>
            )}
          </Space>
        ),
      },
      {
        title: intl.formatMessage({ id: 'perm.col.code' }),
        dataIndex: 'code',
        width: 180,
        render: (code: string) => (
          <Text code style={{ fontSize: 12 }}>
            {code}
          </Text>
        ),
      },
      {
        title: intl.formatMessage({ id: 'perm.col.actionType' }),
        dataIndex: 'action',
        width: 90,
        render: (action: string) => (
          <Tag color={ACTION_COLOR[action] || 'default'}>
            {actionLabel[action as keyof typeof actionLabel] || action}
          </Tag>
        ),
      },
      {
        title: intl.formatMessage({ id: 'perm.col.resource' }),
        dataIndex: 'resource',
        width: 240,
        render: (r: string) => (
          <Text type="secondary" style={{ fontSize: 12 }}>
            {r}
          </Text>
        ),
      },
      {
        title: intl.formatMessage({ id: 'perm.col.desc' }),
        dataIndex: 'descKey',
        ellipsis: true,
        render: (_: string, record: (typeof SYSTEM_PERMISSIONS)[0]) =>
          intl.formatMessage({ id: record.descKey }),
      },
    ],
    [intl, existingCodes, actionLabel]
  );

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'perm.catalog.title' })}
      subTitle={intl.formatMessage({ id: 'perm.catalog.subtitle' }, { count: SYSTEM_PERMISSIONS.length })}
      extra={
        missingCount > 0 ? (
          <Button
            type="primary"
            icon={<SyncOutlined spin={syncing} />}
            loading={syncing}
            onClick={handleSync}
          >
            {intl.formatMessage({ id: 'perm.sync.button' }, { count: missingCount })}
          </Button>
        ) : (
          <Button icon={<CheckCircleOutlined />} disabled>
            {intl.formatMessage({ id: 'perm.sync.allDone' })}
          </Button>
        )
      }
    >
      {missingCount > 0 && (
        <Alert
          type="warning"
          showIcon
          message={intl.formatMessage({ id: 'perm.alert.unsynced' }, { count: missingCount })}
          style={{ marginBottom: 16 }}
        />
      )}

      {PERM_MODULE_ORDER.map((mod: PermModuleKey) => {
        const perms = SYSTEM_PERMISSIONS.filter((p) => p.module === mod);
        if (!perms.length) return null;
        return (
          <div key={mod} style={{ marginBottom: 24 }}>
            <div style={{ marginBottom: 8 }}>
              <Tag
                color={MODULE_COLOR[mod] || 'default'}
                style={{ fontSize: 13, padding: '2px 10px', fontWeight: 500 }}
              >
                {intl.formatMessage({ id: `perm.module.${mod}` })}
              </Tag>
            </div>
            <Table<(typeof SYSTEM_PERMISSIONS)[0]>
              dataSource={perms}
              columns={columns}
              rowKey="code"
              size="small"
              pagination={false}
              showHeader={mod === PERM_MODULE_ORDER[0]}
            />
          </div>
        );
      })}
    </PageContainer>
  );
};
