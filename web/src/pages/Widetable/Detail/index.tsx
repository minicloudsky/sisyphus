import { PageContainer } from '@ant-design/pro-components';
import {
  Button,
  Card,
  Descriptions,
  message,
  Space,
  Spin,
  Table,
  Tag,
  Tabs,
  Input,
  InputNumber,
  Typography,
} from 'antd';
import {
  ArrowLeftOutlined,
  EditOutlined,
  PlayCircleOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import {
  getWidetableConfig,
  previewWidetableSQL,
  queryWidetable,
  syncWidetable,
  listWidetableSyncRecords,
} from '@/services/api';
import { useEffect, useState, useMemo } from 'react';
import { useIntl } from '@/locales';

const { Text } = Typography;

export default () => {
  const intl = useIntl();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const configId = Number(id);

  const [config, setConfig] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [activeTab, setActiveTab] = useState(searchParams.get('tab') || 'info');

  const [queryLoading, setQueryLoading] = useState(false);
  const [queryResult, setQueryResult] = useState<any>(null);
  const [extraCondition, setExtraCondition] = useState('');
  const [orderBy, setOrderBy] = useState('');
  const [queryPage, setQueryPage] = useState(1);
  const [queryPageSize, setQueryPageSize] = useState(50);

  const [syncLoading, setSyncLoading] = useState(false);
  const [syncRecords, setSyncRecords] = useState<any[]>([]);
  const [syncTotal, setSyncTotal] = useState(0);
  const [syncPage, setSyncPage] = useState(1);
  const [batchSize, setBatchSize] = useState(1000);

  const [previewSQL, setPreviewSQL] = useState('');
  const [previewLoading, setPreviewLoading] = useState(false);

  const syncStatusMap = useMemo(
    () =>
      ({
        0: { text: intl.formatMessage({ id: 'widetable.sync.wait' }), color: 'default' },
        1: { text: intl.formatMessage({ id: 'widetable.sync.running' }), color: 'processing' },
        2: { text: intl.formatMessage({ id: 'widetable.sync.ok' }), color: 'success' },
        3: { text: intl.formatMessage({ id: 'widetable.sync.fail' }), color: 'error' },
      }) as Record<number, { text: string; color: string }>,
    [intl]
  );

  useEffect(() => {
    loadConfig();
  }, [configId]);

  useEffect(() => {
    if (activeTab === 'sync') {
      loadSyncRecords();
    }
  }, [activeTab, syncPage]);

  const loadConfig = async () => {
    setLoading(true);
    try {
      const resp: any = await getWidetableConfig(configId);
      const data = resp.data || resp;
      setConfig(data);
    } catch {
    } finally {
      setLoading(false);
    }
  };

  const loadPreviewSQL = async () => {
    setPreviewLoading(true);
    try {
      const resp: any = await previewWidetableSQL(configId, { limit: 10 });
      if (resp.code === 200) {
        setPreviewSQL(resp.data?.sql || resp.sql || '');
      } else {
        message.error(resp.message || intl.formatMessage({ id: 'widetable.previewFail' }));
      }
    } catch {
    } finally {
      setPreviewLoading(false);
    }
  };

  const handleQuery = async () => {
    setQueryLoading(true);
    try {
      const resp: any = await queryWidetable(configId, {
        page: queryPage,
        page_size: queryPageSize,
        extra_condition: extraCondition || undefined,
        order_by: orderBy || undefined,
      });
      if (resp.code === 200) {
        setQueryResult(resp.data || resp);
      } else {
        message.error(resp.message || intl.formatMessage({ id: 'widetable.queryFail' }));
      }
    } catch {
    } finally {
      setQueryLoading(false);
    }
  };

  const handleSync = async () => {
    setSyncLoading(true);
    try {
      const resp: any = await syncWidetable(configId, { batch_size: batchSize });
      if (resp.code === 200) {
        message.success(intl.formatMessage({ id: 'widetable.syncStarted' }));
        loadSyncRecords();
      } else {
        message.error(resp.message || intl.formatMessage({ id: 'widetable.syncStartFail' }));
      }
    } catch {
    } finally {
      setSyncLoading(false);
    }
  };

  const loadSyncRecords = async () => {
    try {
      const resp: any = await listWidetableSyncRecords(configId, {
        page: syncPage,
        page_size: 10,
      });
      if (resp.code === 200) {
        const data = resp.data || resp;
        setSyncRecords(data.list || []);
        setSyncTotal(data.page?.total || data.total || 0);
      }
    } catch {
    }
  };

  const renderQueryResultTable = () => {
    if (!queryResult) return null;
    const columns = (queryResult.columns || []).map((col: string) => ({
      title: col,
      dataIndex: col,
      key: col,
      ellipsis: true,
    }));
    const dataSource = (queryResult.rows || []).map((row: any, idx: number) => {
      const rowValues = row.values || row;
      const obj: any = { _key: idx };
      (queryResult.columns || []).forEach((col: string, i: number) => {
        obj[col] = rowValues[i];
      });
      return obj;
    });
    return (
      <Table
        columns={columns}
        dataSource={dataSource}
        rowKey="_key"
        size="small"
        scroll={{ x: 'max-content', y: 400 }}
        pagination={false}
      />
    );
  };

  const syncColumns = useMemo(
    () => [
      { title: intl.formatMessage({ id: 'common.id' }), dataIndex: 'id', width: 80 },
      {
        title: intl.formatMessage({ id: 'widetable.col.syncType' }),
        dataIndex: 'sync_type',
        width: 100,
        render: (v: number) =>
          v === 0
            ? intl.formatMessage({ id: 'widetable.syncType.full' })
            : intl.formatMessage({ id: 'widetable.syncType.inc' }),
      },
      {
        title: intl.formatMessage({ id: 'instance.col.status' }),
        dataIndex: 'status',
        width: 100,
        render: (v: number) => {
          const s = syncStatusMap[v] || {
            text: intl.formatMessage({ id: 'widetable.unknownStatus' }, { status: v }),
            color: 'default',
          };
          return <Tag color={s.color}>{s.text}</Tag>;
        },
      },
      { title: intl.formatMessage({ id: 'widetable.col.processed' }), dataIndex: 'processed_rows', width: 110 },
      { title: intl.formatMessage({ id: 'widetable.col.costSec' }), dataIndex: 'cost_seconds', width: 100 },
      { title: intl.formatMessage({ id: 'widetable.col.startedAt' }), dataIndex: 'started_at', width: 180 },
      { title: intl.formatMessage({ id: 'widetable.col.finishedAt' }), dataIndex: 'finished_at', width: 180 },
      {
        title: intl.formatMessage({ id: 'widetable.col.errMsg' }),
        dataIndex: 'error_message',
        ellipsis: true,
        render: (v: string) => (v ? <Text type="danger">{v}</Text> : '-'),
      },
    ],
    [intl, syncStatusMap]
  );

  const tabItems = useMemo(
    () => [
      {
        key: 'info',
        label: intl.formatMessage({ id: 'widetable.tab.info' }),
        children: (
          <Card>
            {config && (
              <>
                <Descriptions column={2} bordered size="small">
                  <Descriptions.Item label={intl.formatMessage({ id: 'widetable.desc.id' })}>{config.id}</Descriptions.Item>
                  <Descriptions.Item label={intl.formatMessage({ id: 'widetable.desc.name' })}>{config.name}</Descriptions.Item>
                  <Descriptions.Item label={intl.formatMessage({ id: 'widetable.desc.description' })} span={2}>{config.description || '-'}</Descriptions.Item>
                  <Descriptions.Item label={intl.formatMessage({ id: 'widetable.desc.sourceInst' })}>{config.source_instance_id}</Descriptions.Item>
                  <Descriptions.Item label={intl.formatMessage({ id: 'widetable.desc.mainTable' })}>{config.main_database}.{config.main_table}</Descriptions.Item>
                  <Descriptions.Item label={intl.formatMessage({ id: 'widetable.desc.targetInst' })}>{config.target_instance_id || '-'}</Descriptions.Item>
                  <Descriptions.Item label={intl.formatMessage({ id: 'widetable.desc.targetTable' })}>
                    {config.target_database && config.target_table
                      ? `${config.target_database}.${config.target_table}`
                      : '-'}
                  </Descriptions.Item>
                  <Descriptions.Item label={intl.formatMessage({ id: 'widetable.desc.syncCond' })} span={2}>{config.sync_condition || '-'}</Descriptions.Item>
                  <Descriptions.Item label={intl.formatMessage({ id: 'widetable.desc.created' })}>{config.created_at}</Descriptions.Item>
                  <Descriptions.Item label={intl.formatMessage({ id: 'widetable.desc.updated' })}>{config.updated_at}</Descriptions.Item>
                </Descriptions>

                <div style={{ marginTop: 16 }}>
                  <Button
                    type="default"
                    icon={<PlayCircleOutlined />}
                    loading={previewLoading}
                    onClick={loadPreviewSQL}
                  >
                    {intl.formatMessage({ id: 'widetable.previewSql' })}
                  </Button>
                </div>

                {previewSQL && (
                  <pre
                    style={{
                      marginTop: 12,
                      background: '#1e1e1e',
                      color: '#d4d4d4',
                      padding: 16,
                      borderRadius: 4,
                      overflowX: 'auto',
                      fontFamily: 'monospace',
                      fontSize: 13,
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-all',
                    }}
                  >
                    {previewSQL}
                  </pre>
                )}
              </>
            )}
          </Card>
        ),
      },
      {
        key: 'query',
        label: intl.formatMessage({ id: 'widetable.tab.query' }),
        children: (
          <Card>
            <div
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: 8,
                marginBottom: 12,
                alignItems: 'flex-end',
              }}
            >
              <Input
                placeholder={intl.formatMessage({ id: 'widetable.ph.extraWhere' })}
                value={extraCondition}
                onChange={(e) => setExtraCondition(e.target.value)}
                style={{ flex: '1 1 200px', minWidth: 160, maxWidth: 520 }}
              />
              <Input
                placeholder={intl.formatMessage({ id: 'widetable.ph.orderBy' })}
                value={orderBy}
                onChange={(e) => setOrderBy(e.target.value)}
                style={{ flex: '1 1 160px', minWidth: 140, maxWidth: 360 }}
              />
              <InputNumber
                placeholder={intl.formatMessage({ id: 'widetable.ph.pageSize' })}
                value={queryPageSize}
                min={1}
                max={1000}
                onChange={(v) => setQueryPageSize(v || 50)}
                style={{ width: 112 }}
              />
              <Button
                type="primary"
                icon={<PlayCircleOutlined />}
                loading={queryLoading}
                onClick={handleQuery}
              >
                {intl.formatMessage({ id: 'widetable.runQuery' })}
              </Button>
            </div>

            {queryResult && (
              <div>
                <div style={{ marginBottom: 8 }}>
                  <Text type="secondary">
                    {intl.formatMessage(
                      { id: 'widetable.resultSummary' },
                      { total: queryResult.total || 0, time: queryResult.execute_time || 0 }
                    )}
                  </Text>
                </div>
                {renderQueryResultTable()}
              </div>
            )}
          </Card>
        ),
      },
      {
        key: 'sync',
        label: intl.formatMessage({ id: 'widetable.tab.sync' }),
        children: (
          <Card>
            <div
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: 12,
                marginBottom: 16,
                alignItems: 'center',
              }}
            >
              <Space>
                <Text>{intl.formatMessage({ id: 'widetable.batchSize' })}</Text>
                <InputNumber
                  value={batchSize}
                  min={100}
                  max={10000}
                  step={100}
                  onChange={(v) => setBatchSize(v || 1000)}
                  style={{ width: 120 }}
                />
              </Space>
              <Button
                type="primary"
                icon={<SyncOutlined />}
                loading={syncLoading}
                onClick={handleSync}
              >
                {intl.formatMessage({ id: 'widetable.startSync' })}
              </Button>
              <Button onClick={loadSyncRecords}>{intl.formatMessage({ id: 'widetable.refreshRecords' })}</Button>
            </div>

            <Table
              columns={syncColumns}
              dataSource={syncRecords}
              rowKey="id"
              size="small"
              scroll={{ x: 1000 }}
              pagination={{
                current: syncPage,
                total: syncTotal,
                pageSize: 10,
                onChange: (p) => setSyncPage(p),
              }}
            />
          </Card>
        ),
      },
    ],
    [
      intl,
      config,
      previewSQL,
      previewLoading,
      extraCondition,
      orderBy,
      queryPageSize,
      queryLoading,
      queryResult,
      syncLoading,
      batchSize,
      syncColumns,
      syncRecords,
      syncPage,
      syncTotal,
    ]
  );

  return (
    <PageContainer
      title={
        config
          ? intl.formatMessage({ id: 'widetable.detail.title' }, { name: config.name })
          : intl.formatMessage({ id: 'widetable.detail.titlePlain' })
      }
      extra={[
        <Button
          key="back"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate('/widetable/list')}
        >
          {intl.formatMessage({ id: 'widetable.backList' })}
        </Button>,
        <Button
          key="edit"
          type="primary"
          icon={<EditOutlined />}
          onClick={() => navigate(`/widetable/create?id=${configId}`)}
        >
          {intl.formatMessage({ id: 'widetable.editConfig' })}
        </Button>,
      ]}
    >
      <Spin spinning={loading}>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={tabItems}
        />
      </Spin>
    </PageContainer>
  );
};
