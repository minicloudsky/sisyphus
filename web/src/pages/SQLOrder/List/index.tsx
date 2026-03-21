import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Button, Space, Tag, message, Popconfirm, Modal, Card, Descriptions, Tabs, Table, Input, Tooltip } from 'antd';
import { useNavigate } from 'react-router-dom';
import { EyeOutlined, CheckOutlined, CloseOutlined, PlayCircleOutlined, FileTextOutlined } from '@ant-design/icons';
import { listSQLOrders, approveSQLOrder, rejectSQLOrder, executeSQLOrder, getSQLOrder, getOrderItems, getOrderRecords, listInstances, listUsers } from '@/services/api';
import type { ProColumns } from '@ant-design/pro-components';
import { useRequest } from 'ahooks';
import { useState, useMemo } from 'react';
import { useIntl } from '@/locales';
import AutoHeightSqlEditor from '../components/AutoHeightSqlEditor';
import { formatDateTime } from '@/utils/format';

export default () => {
  const intl = useIntl();
  const navigate = useNavigate();

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
    [intl.locale, intl]
  );

  const orderTypeLabels = useMemo(
    () => [
      intl.formatMessage({ id: 'home.orderType.ddl' }),
      intl.formatMessage({ id: 'home.orderType.dml' }),
      intl.formatMessage({ id: 'home.orderType.export' }),
    ],
    [intl]
  );

  const riskTexts = useMemo(
    () => [
      intl.formatMessage({ id: 'risk.low' }),
      intl.formatMessage({ id: 'risk.mid' }),
      intl.formatMessage({ id: 'risk.high' }),
    ],
    [intl]
  );

  const executeMethodLabels = useMemo(
    () => [
      intl.formatMessage({ id: 'sqlOrder.exec.auto' }),
      'gh-ost',
      'goInception',
    ],
    [intl]
  );

  const itemExecStatus = useMemo(
    () => [
      intl.formatMessage({ id: 'sqlOrder.item.pending' }),
      intl.formatMessage({ id: 'sqlOrder.item.ok' }),
      intl.formatMessage({ id: 'sqlOrder.item.fail' }),
    ],
    [intl]
  );

  const errLevelTexts = useMemo(
    () => [
      intl.formatMessage({ id: 'sqlOrder.level.none' }),
      intl.formatMessage({ id: 'sqlOrder.level.warn' }),
      intl.formatMessage({ id: 'sqlOrder.level.err' }),
    ],
    [intl]
  );

  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedOrderId, setSelectedOrderId] = useState<string>('');
  const [instanceMap, setInstanceMap] = useState<Record<string, string>>({});
  const [userMap, setUserMap] = useState<Record<string, string>>({});

  // 获取实例列表
  const { data: instancesData } = useRequest(() => listInstances({ page: 1, pageSize: 1000 }), {
    onSuccess: (response: any) => {
      const map: Record<string, string> = {};
      const list = response.data?.list || response.list || [];
      list.forEach((inst: any) => {
        map[String(inst.id)] = inst.name || inst.instanceName || inst.host || String(inst.id);
      });
      setInstanceMap(map);
    },
  });

  // 获取用户列表
  const { data: usersData } = useRequest(() => listUsers({ page: 1, pageSize: 1000 }), {
    onSuccess: (response: any) => {
      const map: Record<string, string> = {};
      const list = response.data?.list || response.list || [];
      list.forEach((user: any) => {
        map[String(user.id)] = user.username || user.realName || user.name || String(user.id);
      });
      setUserMap(map);
    },
  });

  // 获取工单详情（用于弹窗）
  const { data: orderDetail, loading: detailLoading, refresh: refreshDetail } = useRequest(
    () => getSQLOrder(selectedOrderId),
    {
      ready: !!selectedOrderId && detailModalVisible,
    }
  );

  const { data: orderItemsData } = useRequest(
    () => getOrderItems(selectedOrderId),
    {
      ready: !!selectedOrderId && detailModalVisible,
    }
  );

  const { data: orderRecordsData } = useRequest(
    () => getOrderRecords(selectedOrderId),
    {
      ready: !!selectedOrderId && detailModalVisible,
    }
  );

  const order = orderDetail?.data;
  const orderItems = orderItemsData?.data || [];
  // 处理审核记录数据
  let records: API.SQLReviewRecord[] = [];
  if (orderRecordsData) {
    records = orderRecordsData.data || [];
  }

  const { run: handleApprove } = useRequest(
    async (id: string) => {
      const response = await approveSQLOrder(id, {});
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'common.approveOk' }));
        if (id === selectedOrderId) {
          refreshDetail();
        }
        return true;
      } else {
        message.error(response.message || intl.formatMessage({ id: 'common.operationFailed' }));
        return false;
      }
    },
    { manual: true }
  );

  const { run: handleReject } = useRequest(
    async (id: string, reason: string) => {
      const response = await rejectSQLOrder(id, { reason });
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'common.rejected' }));
        if (id === selectedOrderId) {
          refreshDetail();
        }
        return true;
      } else {
        message.error(response.message || intl.formatMessage({ id: 'common.operationFailed' }));
        return false;
      }
    },
    { manual: true }
  );

  const { run: handleExecute } = useRequest(
    async (id: string) => {
      const response = await executeSQLOrder(id);
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'common.executeOk' }));
        if (id === selectedOrderId) {
          refreshDetail();
        }
        return true;
      } else {
        message.error(response.message || intl.formatMessage({ id: 'common.executeFail' }));
        return false;
      }
    },
    { manual: true }
  );

  // 打开详情弹窗
  const handleOpenDetail = (id: string) => {
    setSelectedOrderId(id);
    setDetailModalVisible(true);
  };

  const columns: ProColumns<API.SQLOrder>[] = [
    {
      title: intl.formatMessage({ id: 'home.col.workId' }),
      dataIndex: 'workId',
      width: 150,
      fixed: 'left',
      render: (_, record) => (
        <a onClick={() => navigate(`/sql-order/detail/${record.id}`)}>{record.workId}</a>
      ),
    },
    {
      title: intl.formatMessage({ id: 'sqlOrder.col.instance' }),
      dataIndex: 'instanceId',
      width: 150,
      hideInSearch: true,
      render: (_, record) => instanceMap[String(record.instanceId)] || String(record.instanceId),
    },
    {
      title: intl.formatMessage({ id: 'home.col.database' }),
      dataIndex: 'databaseName',
      width: 120,
    },
    {
      title: intl.formatMessage({ id: 'home.col.type' }),
      dataIndex: 'orderType',
      width: 80,
      valueEnum: {
        0: { text: orderTypeLabels[0] },
        1: { text: orderTypeLabels[1] },
        2: { text: orderTypeLabels[2] },
      },
    },
    {
      title: intl.formatMessage({ id: 'home.col.status' }),
      dataIndex: 'status',
      width: 100,
      valueEnum: statusMap,
      render: (_, record) => {
        const status =
          statusMap[record.status] || {
            text: `${intl.formatMessage({ id: 'home.status.unknown' })} (${record.status})`,
            color: 'default',
          };
        return <Tag color={status.color}>{status.text}</Tag>;
      },
    },
    {
      title: intl.formatMessage({ id: 'sqlOrder.col.risk' }),
      dataIndex: 'riskLevel',
      width: 100,
      valueEnum: {
        0: { text: riskTexts[0] },
        1: { text: riskTexts[1] },
        2: { text: riskTexts[2] },
      },
      render: (_, record) => {
        const colors = ['green', 'orange', 'red'];
        const riskLevel = record.riskLevel ?? 0;
        const color = colors[riskLevel] || 'default';
        const text = riskTexts[riskLevel] || intl.formatMessage({ id: 'home.status.unknown' });
        return <Tag color={color}>{text}</Tag>;
      },
    },
    {
      title: intl.formatMessage({ id: 'sqlOrder.col.submitter' }),
      dataIndex: 'submitterId',
      width: 120,
      hideInSearch: true,
      render: (_, record) => userMap[String(record.submitterId)] || String(record.submitterId),
    },
    {
      title: intl.formatMessage({ id: 'home.col.createdAt' }),
      dataIndex: 'createdAt',
      width: 180,
      hideInSearch: true,
      render: (_, record) => formatDateTime(record.createdAt),
    },
    {
      title: intl.formatMessage({ id: 'home.col.action' }),
      valueType: 'option',
      width: 180,
      fixed: 'right',
      render: (_, record) => {
        return (
          <Space>
            <Button
              type="link"
              icon={<FileTextOutlined />}
              onClick={() => handleOpenDetail(record.id)}
            >
              {intl.formatMessage({ id: 'common.detail' })}
            </Button>
            <Button
              type="link"
              icon={<EyeOutlined />}
              onClick={() => navigate(`/sql-order/detail/${record.id}`)}
            >
              {intl.formatMessage({ id: 'common.view' })}
            </Button>
          </Space>
        );
      },
    },
  ];

  return (
    <PageContainer title={intl.formatMessage({ id: 'sqlOrder.manageTitle' })}>
      <ProTable<API.SQLOrder>
        columns={columns}
        request={async (params) => {
          try {
            const response = await listSQLOrders({
              page: params.current || 1,
              pageSize: params.pageSize || 10,
              status: params.status,
              orderType: params.orderType,
              instanceId: params.instanceId,
            });
            
            // 处理不同的响应结构
            let data: API.SQLOrder[] = [];
            let total = 0;

            if (response.code === 200) {
              data = response.data?.list || [];
              total = response.data?.page?.total || data.length;
            }
            
            return {
              data: data,
              success: response.code === 200,
              total: total,
            };
          } catch (error) {
            console.error(intl.formatMessage({ id: 'sqlOrder.fetchListFail' }), error);
            return {
              data: [],
              success: false,
              total: 0,
            };
          }
        }}
        rowKey="id"
        search={{
          labelWidth: 'auto',
        }}
        pagination={{
          defaultPageSize: 10,
          showSizeChanger: true,
        }}
        scroll={{ x: 1200 }}
      />

      {/* 详情弹窗 */}
      <Modal
        title={intl.formatMessage({ id: 'sqlOrder.modalTitle' })}
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        width={1200}
        footer={null}
        destroyOnClose
      >
        {order && (
          <>
            <Card style={{ marginBottom: 16 }}>
              <Descriptions column={2} bordered>
                <Descriptions.Item label={intl.formatMessage({ id: 'home.col.workId' })}>
                  {order.workId}
                </Descriptions.Item>
                <Descriptions.Item label={intl.formatMessage({ id: 'home.col.status' })}>
                  {(() => {
                    const status =
                      statusMap[order.status] || {
                        text: `${intl.formatMessage({ id: 'home.status.unknown' })} (${order.status})`,
                        color: 'default',
                      };
                    return <Tag color={status.color}>{status.text}</Tag>;
                  })()}
                </Descriptions.Item>
                <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.col.instance' })}>
                  {instanceMap[order.instanceId] || order.instanceId}
                </Descriptions.Item>
                <Descriptions.Item label={intl.formatMessage({ id: 'home.col.database' })}>
                  {order.databaseName}
                </Descriptions.Item>
                <Descriptions.Item label={intl.formatMessage({ id: 'home.col.type' })}>
                  {orderTypeLabels[order.orderType] || `?(${order.orderType})`}
                </Descriptions.Item>
                <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.col.risk' })}>
                  {(() => {
                    const riskLevel = order.riskLevel ?? 0;
                    const colors = ['green', 'orange', 'red'];
                    const color = colors[riskLevel] || 'default';
                    const text = riskTexts[riskLevel] || intl.formatMessage({ id: 'home.status.unknown' });
                    return <Tag color={color}>{text}</Tag>;
                  })()}
                </Descriptions.Item>
                <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.backupNeed' })}>
                  {order.needBackup
                    ? intl.formatMessage({ id: 'common.yes' })
                    : intl.formatMessage({ id: 'common.no' })}
                </Descriptions.Item>
                <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.execMethod' })}>
                  {executeMethodLabels[order.executeMethod] || `?(${order.executeMethod})`}
                </Descriptions.Item>
                <Descriptions.Item label={intl.formatMessage({ id: 'home.col.createdAt' })}>
                  {formatDateTime(order.createdAt)}
                </Descriptions.Item>
                {order.remark && (
                  <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.remark' })} span={2}>
                    {order.remark}
                  </Descriptions.Item>
                )}
                {order.rejectReason && (
                  <Descriptions.Item label={intl.formatMessage({ id: 'sqlOrder.rejectReason' })} span={2}>
                    <span style={{ color: '#ff4d4f' }}>{order.rejectReason}</span>
                  </Descriptions.Item>
                )}
              </Descriptions>
            </Card>

            <Card>
              <Tabs
                items={[
                  {
                    key: 'sql',
                    label: intl.formatMessage({ id: 'sqlOrder.tab.sql' }),
                    children: (
                      <AutoHeightSqlEditor value={order.sqlContent || ''} minHeight={120} maxHeight={560} />
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
                            render: (seq) => seq || '-'
                          },
                          { 
                            title: intl.formatMessage({ id: 'sqlOrder.col.sql' }), 
                            dataIndex: 'sqlText', 
                            ellipsis: { showTitle: false },
                            render: (text) => {
                              if (!text || text.trim() === '') {
                                return (
                                  <span style={{ color: '#999' }}>
                                    {intl.formatMessage({ id: 'sqlOrder.noSql' })}
                                  </span>
                                );
                              }
                              return (
                                <Tooltip title={text}>
                                  <span style={{ fontFamily: 'monospace', fontSize: '12px' }}>{text}</span>
                                </Tooltip>
                              );
                            }
                          },
                          {
                            title: intl.formatMessage({ id: 'home.col.status' }),
                            dataIndex: 'status',
                            width: 100,
                            render: (status) => {
                              const colors = ['orange', 'green', 'red'];
                              const statusNum = status ?? 0;
                              return (
                                <Tag color={colors[statusNum] || 'default'}>
                                  {itemExecStatus[statusNum] ||
                                    intl.formatMessage({ id: 'home.status.unknown' })}
                                </Tag>
                              );
                            },
                          },
                          { 
                            title: intl.formatMessage({ id: 'sqlOrder.col.rows' }), 
                            dataIndex: 'affectedRows', 
                            width: 100,
                            render: (rows) => rows ?? 0
                          },
                          {
                            title: intl.formatMessage({ id: 'sqlOrder.col.duration' }),
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
                            }
                          },
                        ]}
                        dataSource={orderItems}
                        rowKey="id"
                        pagination={false}
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
                            render: (seq) => seq || '-'
                          },
                          { 
                            title: intl.formatMessage({ id: 'sqlOrder.col.sql' }), 
                            dataIndex: 'sqlText', 
                            ellipsis: { showTitle: false },
                            render: (text) => {
                              if (!text || text.trim() === '') {
                                return (
                                  <span style={{ color: '#999' }}>
                                    {intl.formatMessage({ id: 'sqlOrder.noSql' })}
                                  </span>
                                );
                              }
                              return (
                                <Tooltip title={text}>
                                  <span style={{ fontFamily: 'monospace' }}>{text}</span>
                                </Tooltip>
                              );
                            }
                          },
                          {
                            title: intl.formatMessage({ id: 'sqlOrder.col.errLevel' }),
                            dataIndex: 'errorLevel',
                            width: 100,
                            render: (level) => {
                              const levelNum = level ?? 0;
                              const colors = ['green', 'orange', 'red'];
                              return (
                                <Tag color={colors[levelNum] || 'default'}>
                                  {errLevelTexts[levelNum] ||
                                    `${intl.formatMessage({ id: 'home.status.unknown' })} (${levelNum})`}
                                </Tag>
                              );
                            },
                          },
                          {
                            title: intl.formatMessage({ id: 'home.col.status' }),
                            dataIndex: 'stageStatus',
                            width: 120,
                            render: (status) => {
                              if (!status) {
                                return (
                                  <Tag color="default">
                                    {intl.formatMessage({ id: 'home.status.unknown' })}
                                  </Tag>
                                );
                              }
                              const statusText = statusMap[status] || { text: status, color: 'default' };
                              return <Tag color={statusText.color}>{statusText.text}</Tag>;
                            },
                          },
                          {
                            title: intl.formatMessage({ id: 'sqlOrder.col.errMsg' }),
                            dataIndex: 'errorMessage',
                            ellipsis: { showTitle: false },
                            render: (msg) => {
                              if (!msg || msg.trim() === '') {
                                return (
                                  <span style={{ color: '#999' }}>
                                    {intl.formatMessage({ id: 'sqlOrder.noErrInfo' })}
                                  </span>
                                );
                              }
                              const displayText = msg.length > 50 ? msg.substring(0, 50) + '...' : msg;
                              return (
                                <Tooltip title={msg}>
                                  <span>{displayText}</span>
                                </Tooltip>
                              );
                            }
                          },
                        ]}
                        dataSource={records}
                        rowKey="id"
                        pagination={false}
                      />
                    ),
                  },
                ]}
              />
            </Card>

            {/* 操作按钮 */}
            <div style={{ marginTop: 16, textAlign: 'right' }}>
              <Space>
                {order.status === 0 && (
                  <>
                    <Popconfirm
                      title={intl.formatMessage({ id: 'sqlOrder.confirmApprove' })}
                      onConfirm={() => {
                        handleApprove(order.id);
                      }}
                    >
                      <Button type="primary" icon={<CheckOutlined />}>
                        {intl.formatMessage({ id: 'sqlOrder.approve' })}
                      </Button>
                    </Popconfirm>
                    <Button
                      danger
                      icon={<CloseOutlined />}
                      onClick={() => {
                        let rejectReason = '';
                        Modal.confirm({
                          title: intl.formatMessage({ id: 'sqlOrder.rejectTitle' }),
                          content: (
                            <Input.TextArea
                              rows={4}
                              placeholder={intl.formatMessage({ id: 'sqlOrder.rejectPlaceholder' })}
                              onChange={(e) => {
                                rejectReason = e.target.value;
                              }}
                            />
                          ),
                          onOk: () => {
                            if (rejectReason.trim()) {
                              handleReject(order.id, rejectReason.trim());
                            } else {
                              message.warning(intl.formatMessage({ id: 'sqlOrder.rejectWarn' }));
                              return Promise.reject();
                            }
                          },
                        });
                      }}
                    >
                      {intl.formatMessage({ id: 'sqlOrder.reject' })}
                    </Button>
                  </>
                )}
                {order.status === 1 && (
                  <Popconfirm
                    title={intl.formatMessage({ id: 'sqlOrder.confirmExecute' })}
                    onConfirm={() => {
                      handleExecute(order.id);
                    }}
                  >
                    <Button type="primary" icon={<PlayCircleOutlined />}>
                      {intl.formatMessage({ id: 'sqlOrder.execute' })}
                    </Button>
                  </Popconfirm>
                )}
                <Button onClick={() => navigate(`/sql-order/detail/${order.id}`)}>
                  {intl.formatMessage({ id: 'sqlOrder.viewFull' })}
                </Button>
              </Space>
            </div>
          </>
        )}
        {detailLoading && (
          <div style={{ textAlign: 'center', padding: 50 }}>
            {intl.formatMessage({ id: 'common.loading' })}
          </div>
        )}
      </Modal>
    </PageContainer>
  );
};
