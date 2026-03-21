import { PageContainer } from '@ant-design/pro-components';
import { Card, Descriptions, Tag, Tabs, Table, Tooltip, Button, Space, Popconfirm, Modal, Input, message } from 'antd';
import { CheckOutlined, CloseOutlined, PlayCircleOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useParams } from 'react-router-dom';
import { useRequest } from 'ahooks';
import { getSQLOrder, getOrderItems, getOrderRecords, approveSQLOrder, rejectSQLOrder, executeSQLOrder } from '@/services/api';
import AutoHeightSqlEditor from '../components/AutoHeightSqlEditor';
import { formatDateTime } from '@/utils/format';
import { useIntl } from '@/locales';
import { useMemo } from 'react';
import './index.less';

export default () => {
  const intl = useIntl();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const { data, loading } = useRequest(() => getSQLOrder(id!), {
    ready: !!id,
  });
  const { data: itemsData } = useRequest(() => getOrderItems(id!), {
    ready: !!id,
  });
  const { data: recordsData } = useRequest(() => getOrderRecords(id!), {
    ready: !!id,
  });

  const order = data?.data;
  const orderItems = itemsData?.data || [];
  let records: API.SQLReviewRecord[] = [];
  if (recordsData) {
    records = recordsData.data || [];
  }

  const execMethodLabel = useMemo(
    () => (m: number) => {
      const labels = [
        intl.formatMessage({ id: 'sqlOrder.exec.auto' }),
        'gh-ost',
        'goInception',
        intl.formatMessage({ id: 'sqlOrder.detail.exec.ckcheck' }),
      ];
      return labels[m] ?? String(m);
    },
    [intl]
  );

  const { run: handleApprove, loading: approveLoading } = useRequest(
    async () => {
      const response = await approveSQLOrder(id!, {});
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'sqlOrder.detail.approveOk' }));
        window.location.reload();
      } else {
        message.error(response.message || intl.formatMessage({ id: 'common.operationFailed' }));
      }
    },
    { manual: true }
  );

  const { run: handleExecute, loading: executeLoading } = useRequest(
    async () => {
      const response = await executeSQLOrder(id!);
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'sqlOrder.detail.submitExec' }));
        window.location.reload();
      } else {
        message.error(response.message || intl.formatMessage({ id: 'sqlOrder.detail.execFail' }));
      }
    },
    { manual: true }
  );

  const handleReject = () => {
    let rejectReason = '';
    Modal.confirm({
      title: intl.formatMessage({ id: 'sqlOrder.rejectTitle' }),
      content: (
        <Input.TextArea
          rows={4}
          placeholder={intl.formatMessage({ id: 'sqlOrder.rejectPlaceholder' })}
          onChange={(e) => { rejectReason = e.target.value; }}
        />
      ),
      onOk: async () => {
        if (!rejectReason.trim()) {
          message.warning(intl.formatMessage({ id: 'sqlOrder.rejectWarn' }));
          return Promise.reject();
        }
        const response = await rejectSQLOrder(id!, { reason: rejectReason.trim() });
        if (response.code === 200) {
          message.success(intl.formatMessage({ id: 'sqlOrder.detail.rejectOk' }));
          window.location.reload();
        } else {
          message.error(response.message || intl.formatMessage({ id: 'common.operationFailed' }));
        }
      },
    });
  };

  const statusMap = useMemo(
    () =>
      ({
        0: { text: intl.formatMessage({ id: 'home.status.pending' }), color: 'orange' },
        1: { text: intl.formatMessage({ id: 'home.status.approved' }), color: 'green' },
        2: { text: intl.formatMessage({ id: 'home.status.rejected' }), color: 'red' },
        3: { text: intl.formatMessage({ id: 'home.status.executing' }), color: 'blue' },
        4: { text: intl.formatMessage({ id: 'home.status.success' }), color: 'success' },
        5: { text: intl.formatMessage({ id: 'home.status.failed' }), color: 'error' },
        6: { text: intl.formatMessage({ id: 'home.status.review' }), color: 'warning' },
      }) as Record<number, { text: string; color: string }>,
    [intl]
  );

  if (!order) {
    return (
      <PageContainer loading={loading}>
        {intl.formatMessage({ id: 'sqlOrder.detail.notFound' })}
      </PageContainer>
    );
  }

  const tabsItems = [
    {
      key: 'sql',
      label: intl.formatMessage({ id: 'sqlOrder.tab.sql' }),
      children: (
        <AutoHeightSqlEditor value={order.sqlContent || ''} minHeight={120} maxHeight={640} />
      ),
    },
    {
      key: 'items',
      label: intl.formatMessage({ id: 'sqlOrder.tab.items' }),
      children: (
        <Table
          columns={[
            {
              title: intl.formatMessage({ id: 'sqlOrder.col.seq' }),
              dataIndex: 'sequence',
              width: 80,
              render: (seq) => seq || '-',
            },
            {
              title: intl.formatMessage({ id: 'sqlOrder.col.sql' }),
              dataIndex: 'sqlText',
              ellipsis: { showTitle: false },
              render: (text) => {
                if (!text || text.trim() === '') {
                  return <span style={{ color: '#999' }}>{intl.formatMessage({ id: 'sqlOrder.noSql' })}</span>;
                }
                return (
                  <Tooltip title={text}>
                    <span style={{ fontFamily: 'monospace', fontSize: '12px' }}>{text}</span>
                  </Tooltip>
                );
              },
            },
            {
              title: intl.formatMessage({ id: 'home.col.status' }),
              dataIndex: 'status',
              width: 100,
              render: (status) => {
                const colors = ['orange', 'green', 'red'];
                const statusNum = status ?? 0;
                const labels = ['sqlOrder.item.pending', 'sqlOrder.item.ok', 'sqlOrder.item.fail'] as const;
                const label =
                  statusNum >= 0 && statusNum <= 2
                    ? intl.formatMessage({ id: labels[statusNum] })
                    : intl.formatMessage({ id: 'sqlOrder.item.statusUnknown' }, { n: statusNum });
                return <Tag color={colors[statusNum] || 'default'}>{label}</Tag>;
              },
            },
            {
              title: intl.formatMessage({ id: 'sqlOrder.col.rows' }),
              dataIndex: 'affectedRows',
              width: 100,
              render: (rows) => rows ?? 0,
            },
            {
              title: intl.formatMessage({ id: 'sqlOrder.detail.execTime' }),
              dataIndex: 'executeTime',
              width: 120,
              render: (time) => time || '-',
            },
            {
              title: intl.formatMessage({ id: 'sqlOrder.col.errMsg' }),
              dataIndex: 'errorMessage',
              ellipsis: { showTitle: false },
              render: (msg) => {
                if (!msg || msg.trim() === '') {
                  return <span style={{ color: '#999' }}>-</span>;
                }
                const displayText = msg.length > 100 ? msg.substring(0, 100) + '...' : msg;
                return (
                  <Tooltip title={msg}>
                    <span style={{ color: '#ff4d4f' }}>{displayText}</span>
                  </Tooltip>
                );
              },
            },
          ]}
          dataSource={orderItems}
          rowKey="id"
        />
      ),
    },
    {
      key: 'records',
      label: intl.formatMessage({ id: 'sqlOrder.tab.records' }),
      children: (
        <Table
          columns={[
            {
              title: intl.formatMessage({ id: 'sqlOrder.col.seq' }),
              dataIndex: 'sequence',
              width: 80,
              render: (seq) => seq || '-',
            },
            {
              title: intl.formatMessage({ id: 'sqlOrder.col.sql' }),
              dataIndex: 'sqlText',
              ellipsis: { showTitle: false },
              render: (text) => {
                if (!text || text.trim() === '') {
                  return <span style={{ color: '#999' }}>{intl.formatMessage({ id: 'sqlOrder.noSql' })}</span>;
                }
                return (
                  <Tooltip title={text}>
                    <span style={{ fontFamily: 'monospace' }}>{text}</span>
                  </Tooltip>
                );
              },
            },
            {
              title: intl.formatMessage({ id: 'sqlOrder.col.errLevel' }),
              dataIndex: 'errorLevel',
              width: 100,
              render: (level) => {
                const levelNum = level ?? 0;
                const levelText = [
                  intl.formatMessage({ id: 'sqlOrder.level.none' }),
                  intl.formatMessage({ id: 'sqlOrder.level.warn' }),
                  intl.formatMessage({ id: 'sqlOrder.level.err' }),
                ];
                const colors = ['green', 'orange', 'red'];
                return <Tag color={colors[levelNum] || 'default'}>{levelText[levelNum]}</Tag>;
              },
            },
            {
              title: intl.formatMessage({ id: 'sqlOrder.col.errMsg' }),
              dataIndex: 'errorMessage',
              ellipsis: { showTitle: false },
              render: (msg) => {
                if (!msg || msg.trim() === '') {
                  return '-';
                }
                const displayText = msg.length > 50 ? msg.substring(0, 50) + '...' : msg;
                return (
                  <Tooltip title={msg}>
                    <span>{displayText}</span>
                  </Tooltip>
                );
              },
            },
            {
              title: intl.formatMessage({ id: 'sqlOrder.detail.stageStatus' }),
              dataIndex: 'stageStatus',
              width: 120,
              render: (status) => {
                if (!status || status.trim() === '') {
                  return <Tag color="default">{intl.formatMessage({ id: 'sqlOrder.detail.unknown' })}</Tag>;
                }
                const sl = status.toLowerCase();
                let color: string = 'default';
                if (
                  /完成|成功|已审核|complete|success|checked|executed|audit completed|execute successfully/.test(
                    sl
                  )
                ) {
                  color = 'success';
                } else if (
                  /审核中|执行中|备份中|auditing|executing|backing|pending|waiting|progress/.test(sl)
                ) {
                  color = 'processing';
                } else if (/失败|错误|fail|error/.test(sl)) {
                  color = 'error';
                } else if (/警告|回滚|warn|rollback|rolling/.test(sl)) {
                  color = 'warning';
                }
                return <Tag color={color}>{status}</Tag>;
              },
            },
            {
              title: intl.formatMessage({ id: 'sqlOrder.col.rows' }),
              dataIndex: 'affectedRows',
              width: 100,
              render: (rows) => rows ?? 0,
            },
            {
              title: intl.formatMessage({ id: 'sqlOrder.detail.execTime' }),
              dataIndex: 'executeTime',
              width: 120,
              render: (time) => time || '-',
            },
          ]}
          dataSource={records}
          rowKey="id"
        />
      ),
    },
  ];

  const orderTypeLabel = (t: number) => {
    const types = ['DDL', 'DML', intl.formatMessage({ id: 'sqlOrder.create.export' })];
    return types[t] ?? intl.formatMessage({ id: 'sqlOrder.detail.unknownOrderType' }, { n: t });
  };

  const riskLabel = (r: number) => {
    const texts = [intl.formatMessage({ id: 'risk.low' }), intl.formatMessage({ id: 'risk.mid' }), intl.formatMessage({ id: 'risk.high' })];
    return texts[r] ?? intl.formatMessage({ id: 'sqlOrder.detail.unknownRisk' }, { n: r });
  };

  return (
    <PageContainer title={intl.formatMessage({ id: 'sqlOrder.detail.pageTitle' })} onBack={() => navigate(-1)} loading={loading}>
      <Card>
        <Descriptions column={2} bordered>
          <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.detail.label.workId' })}>{order.workId}</Descriptions.Item>
          <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.detail.label.status' })}>
            {(() => {
              const status = statusMap[order.status] || {
                text: intl.formatMessage({ id: 'sqlOrder.item.statusUnknown' }, { n: order.status }),
                color: 'default',
              };
              return <Tag color={status.color}>{status.text}</Tag>;
            })()}
          </Descriptions.Item>
          <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.detail.label.db' })}>{order.databaseName}</Descriptions.Item>
          <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.detail.label.orderType' })}>
            {orderTypeLabel(order.orderType)}
          </Descriptions.Item>
          <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.detail.label.risk' })}>
            {(() => {
              const riskLevel = order.riskLevel ?? 0;
              const colors = ['green', 'orange', 'red'];
              const color = colors[riskLevel] || 'default';
              const text = riskLabel(riskLevel);
              return <Tag color={color}>{text}</Tag>;
            })()}
          </Descriptions.Item>
          <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.detail.label.backup' })}>
            {order.needBackup ? intl.formatMessage({ id: 'common.yes' }) : intl.formatMessage({ id: 'common.no' })}
          </Descriptions.Item>
          <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.detail.label.execMethod' })}>
            {execMethodLabel(order.executeMethod ?? 0)}
          </Descriptions.Item>
          <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.detail.label.submittedAt' })}>
            {order.createdAt ? formatDateTime(order.createdAt) : '-'}
          </Descriptions.Item>
          {order.remark && <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.remark' })} span={2}>{order.remark}</Descriptions.Item>}
          {order.rejectReason && (
            <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.detail.label.rejectReason' })} span={2}>
              <span style={{ color: '#ff4d4f' }}>{order.rejectReason}</span>
            </Descriptions.Item>
          )}
        </Descriptions>
      </Card>

      <Card style={{ marginTop: 16 }}>
        <Tabs items={tabsItems} />
      </Card>

      <Card style={{ marginTop: 16 }}>
        <Space>
          {order.status === 0 && (
            <>
              <Popconfirm title={intl.formatMessage({ id: 'sqlOrder.confirmApprove' })} onConfirm={handleApprove}>
                <Button type="primary" icon={<CheckOutlined />} loading={approveLoading}>
                  {intl.formatMessage({ id: 'sqlOrder.approve' })}
                </Button>
              </Popconfirm>
              <Button danger icon={<CloseOutlined />} onClick={handleReject}>
                {intl.formatMessage({ id: 'sqlOrder.reject' })}
              </Button>
            </>
          )}
          {order.status === 1 && (
            <Popconfirm title={intl.formatMessage({ id: 'sqlOrder.confirmExecute' })} onConfirm={handleExecute}>
              <Button type="primary" icon={<PlayCircleOutlined />} loading={executeLoading}>
                {intl.formatMessage({ id: 'sqlOrder.execute' })}
              </Button>
            </Popconfirm>
          )}
        </Space>
      </Card>
    </PageContainer>
  );
};
