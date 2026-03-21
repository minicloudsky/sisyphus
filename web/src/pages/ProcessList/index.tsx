import { PageContainer } from '@ant-design/pro-components';
import { Card, Table, Button, Select, Space, Typography, Tag, Popconfirm, message, Badge, Tooltip } from 'antd';
import { ReloadOutlined, StopOutlined, DatabaseOutlined } from '@ant-design/icons';
import { listInstances, getProcessList, killProcess } from '@/services/api';
import { useRequest } from 'ahooks';
import { useState, useCallback, useMemo } from 'react';
import { useIntl } from '@/locales';

const { Text } = Typography;

const formatTime = (seconds: number) => {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
};

export default () => {
  const intl = useIntl();
  const [instanceId, setInstanceId] = useState<string>('');
  const [processes, setProcesses] = useState<any[]>([]);
  const [loadingKill, setLoadingKill] = useState<Record<string, boolean>>({});

  const { data: instancesData } = useRequest(() =>
    listInstances({ page: 1, pageSize: 1000, instancePermissionScope: 'query' }),
  );
  const instances = instancesData?.list || [];

  const { loading, run: fetchProcesses } = useRequest(() => getProcessList(instanceId), {
    manual: true,
    onSuccess: (res) => {
      if (res?.code === 200) {
        setProcesses(res.processes || []);
      } else {
        message.error(res?.message || intl.formatMessage({ id: 'process.fetchFail' }));
      }
    },
    onError: () => {},
  });

  const handleKill = useCallback(
    async (processId: string) => {
      setLoadingKill((prev) => ({ ...prev, [processId]: true }));
      try {
        const res = await killProcess(instanceId, processId);
        if (res?.code === 200) {
          message.success(intl.formatMessage({ id: 'process.killSent' }));
          fetchProcesses();
        } else {
          message.error(res?.message || intl.formatMessage({ id: 'process.killFail' }));
        }
      } catch {
        /* ignore */
      } finally {
        setLoadingKill((prev) => ({ ...prev, [processId]: false }));
      }
    },
    [instanceId, fetchProcesses, intl]
  );

  const columns = useMemo(
    () => [
      { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
      {
        title: intl.formatMessage({ id: 'process.col.user' }),
        dataIndex: 'user',
        key: 'user',
        width: 100,
      },
      {
        title: intl.formatMessage({ id: 'process.col.source' }),
        dataIndex: 'host',
        key: 'host',
        width: 160,
        ellipsis: true,
      },
      {
        title: intl.formatMessage({ id: 'process.col.db' }),
        dataIndex: 'db',
        key: 'db',
        width: 120,
        render: (db: string) => db || <Text type="secondary">-</Text>,
      },
      {
        title: intl.formatMessage({ id: 'process.col.command' }),
        dataIndex: 'command',
        key: 'command',
        width: 90,
        render: (cmd: string) => <Tag color={cmd === 'Query' ? 'blue' : 'default'}>{cmd}</Tag>,
      },
      {
        title: intl.formatMessage({ id: 'process.col.elapsed' }),
        dataIndex: 'time',
        key: 'time',
        width: 100,
        sorter: (a: any, b: any) => a.time - b.time,
        render: (t: number) => {
          const color = t > 30 ? 'red' : t > 5 ? 'orange' : 'default';
          return <Tag color={color}>{formatTime(t)}</Tag>;
        },
      },
      {
        title: intl.formatMessage({ id: 'process.col.state' }),
        dataIndex: 'state',
        key: 'state',
        width: 120,
        ellipsis: true,
        render: (state: string, record: any) => {
          if (record.progress && record.progress !== '0.00%') {
            return (
              <span>
                {state || 'Running'} <Badge color="processing" /> {record.progress}
              </span>
            );
          }
          return state || <Text type="secondary">-</Text>;
        },
      },
      {
        title: intl.formatMessage({ id: 'process.col.sql' }),
        dataIndex: 'info',
        key: 'info',
        ellipsis: true,
        render: (info: string) => (
          <Tooltip title={info} overlayStyle={{ maxWidth: 600 }}>
            <Text code style={{ fontSize: 12 }}>
              {info || '-'}
            </Text>
          </Tooltip>
        ),
      },
      {
        title: intl.formatMessage({ id: 'process.col.action' }),
        key: 'action',
        width: 80,
        fixed: 'right' as const,
        render: (_: any, record: any) => (
          <Popconfirm
            title={intl.formatMessage({ id: 'process.killConfirm' }, { id: record.id })}
            description={intl.formatMessage({ id: 'process.killDesc' })}
            onConfirm={() => handleKill(record.query_id || record.id)}
            okText={intl.formatMessage({ id: 'common.confirm' })}
            cancelText={intl.formatMessage({ id: 'common.cancel' })}
            okButtonProps={{ danger: true }}
          >
            <Button
              type="link"
              danger
              size="small"
              icon={<StopOutlined />}
              loading={loadingKill[record.query_id || record.id]}
            >
              {intl.formatMessage({ id: 'process.kill' })}
            </Button>
          </Popconfirm>
        ),
      },
    ],
    [intl, handleKill, loadingKill]
  );

  return (
    <PageContainer title={intl.formatMessage({ id: 'process.title' })}>
      <Card>
        <Space wrap size={[8, 8]} style={{ marginBottom: 16, width: '100%' }}>
          <DatabaseOutlined style={{ color: '#1677ff' }} />
          <Select
            style={{ width: 280, minWidth: 200, maxWidth: '100%' }}
            placeholder={intl.formatMessage({ id: 'process.selectInstance' })}
            value={instanceId || undefined}
            onChange={setInstanceId}
            showSearch
            options={instances.map((inst: any) => ({
              label: `${inst.name} (${inst.dbType || inst.db_type}) - ${inst.host}:${inst.port}`,
              value: String(inst.id),
            }))}
          />
          <Button
            type="primary"
            icon={<ReloadOutlined />}
            onClick={fetchProcesses}
            loading={loading}
            disabled={!instanceId}
          >
            {intl.formatMessage({ id: 'process.refresh' })}
          </Button>
          {processes.length > 0 && (
            <Text type="secondary">
              {intl.formatMessage({ id: 'process.activeCount' }, { count: processes.length })}
            </Text>
          )}
        </Space>

        <Table
          columns={columns}
          dataSource={processes}
          rowKey={(record) => record.query_id || record.id}
          scroll={{ x: 'max-content' }}
          size="small"
          loading={loading}
          pagination={{ pageSize: 50, showSizeChanger: true }}
          locale={{
            emptyText: instanceId
              ? intl.formatMessage({ id: 'process.emptyNoRows' })
              : intl.formatMessage({ id: 'process.emptyNoInstance' }),
          }}
          rowClassName={(record) => (record.time > 30 ? 'process-slow' : '')}
        />
      </Card>
    </PageContainer>
  );
};
