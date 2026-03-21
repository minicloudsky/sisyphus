import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Button, Space, Tag, message, Modal, Form, Select, DatePicker, Table, Tooltip } from 'antd';
import { PlusOutlined, EyeOutlined } from '@ant-design/icons';
import {
  parseBinlog,
  listBinlogOrders,
  getBinlogOrder,
  getBinlogOrderSqls,
  listInstances,
  listDatabases,
  listTables,
} from '@/services/api';
import { useRequest } from 'ahooks';
import type { ProColumns } from '@ant-design/pro-components';
import { useState, useMemo } from 'react';
import { useIntl } from '@/locales';

const { Option } = Select;
const { RangePicker } = DatePicker;

export default () => {
  const intl = useIntl();
  const statusMap = useMemo(
    () =>
      ({
        0: { text: intl.formatMessage({ id: 'binlog.status.parsing' }), color: 'blue' },
        1: { text: intl.formatMessage({ id: 'binlog.status.ok' }), color: 'green' },
        2: { text: intl.formatMessage({ id: 'binlog.status.fail' }), color: 'red' },
      }) as Record<number, { text: string; color: string }>,
    [intl]
  );

  const [parseModalVisible, setParseModalVisible] = useState(false);
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [currentOrder, setCurrentOrder] = useState<API.BinlogOrder | null>(null);
  const [currentOrderSqls, setCurrentOrderSqls] = useState<API.BinlogRollbackSql[]>([]);
  const [form] = Form.useForm();
  const [selectedInstanceId, setSelectedInstanceId] = useState<string>('');
  const [selectedDatabase, setSelectedDatabase] = useState<string>('');

  const { data: instancesData } = useRequest(() =>
    listInstances({ page: 1, pageSize: 1000, dbType: 'mysql' })
  );
  const instances = instancesData?.list || [];

  const {
    data: databasesData,
    loading: databasesLoading,
    run: loadDatabases,
  } = useRequest((instanceId: string) => listDatabases(instanceId), {
    manual: true,
    onError: () => {},
  });
  const databases = databasesData?.databases || [];

  const {
    data: tablesData,
    loading: tablesLoading,
    run: loadTables,
  } = useRequest(
    ({ instanceId, database }: { instanceId: string; database: string }) =>
      listTables(instanceId, database),
    {
      manual: true,
      onError: () => {},
    }
  );
  const tables = tablesData?.tables || [];

  const handleInstanceChange = (instanceId: string) => {
    setSelectedInstanceId(instanceId);
    setSelectedDatabase('');
    form.setFieldsValue({ databaseName: '', tableName: '' });
    if (instanceId) {
      loadDatabases(instanceId);
    }
  };

  const handleDatabaseChange = (database: string) => {
    setSelectedDatabase(database);
    form.setFieldsValue({ tableName: '' });
    if (selectedInstanceId && database) {
      loadTables({ instanceId: selectedInstanceId, database });
    }
  };

  const { run: handleParse, loading: parseLoading } = useRequest(
    async (values: any) => {
      const [startTime, endTime] = values.timeRange;
      const response = await parseBinlog({
        instanceId: values.instanceId,
        databaseName: values.databaseName,
        tableName: values.tableName,
        binlogFile: values.binlogFile || '',
        startTime: startTime.format('YYYY-MM-DD HH:mm:ss'),
        endTime: endTime.format('YYYY-MM-DD HH:mm:ss'),
        remark: values.remark,
      });
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'binlog.submitOk' }));
        setParseModalVisible(false);
        form.resetFields();
        setSelectedInstanceId('');
        setSelectedDatabase('');
        return true;
      } else {
        message.error(response.message || intl.formatMessage({ id: 'binlog.submitFail' }));
        return false;
      }
    },
    { manual: true }
  );

  const { run: loadDetail, loading: detailLoading } = useRequest(
    async (id: string) => {
      const [orderResp, sqlsResp] = await Promise.all([
        getBinlogOrder(id),
        getBinlogOrderSqls(id),
      ]);
      if (orderResp.code === 200 && orderResp.data) {
        setCurrentOrder(orderResp.data);
        const sqls =
          sqlsResp.list || sqlsResp.data || [];
        setCurrentOrderSqls(sqls);
        setDetailModalVisible(true);
      } else {
        message.error(orderResp.message || intl.formatMessage({ id: 'binlog.detailFail' }));
      }
    },
    { manual: true }
  );

  const columns: ProColumns<API.BinlogOrder>[] = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'home.col.workId' }),
        dataIndex: 'workId',
        width: 150,
        fixed: 'left',
      },
      {
        title: intl.formatMessage({ id: 'binlog.list.instanceCol' }),
        dataIndex: 'instanceId',
        width: 100,
        hideInSearch: true,
      },
      {
        title: intl.formatMessage({ id: 'home.col.database' }),
        dataIndex: 'databaseName',
        width: 120,
      },
      {
        title: intl.formatMessage({ id: 'binlog.list.tableCol' }),
        dataIndex: 'tableName',
        width: 120,
      },
      {
        title: intl.formatMessage({ id: 'binlog.list.startTime' }),
        dataIndex: 'startTime',
        width: 180,
        valueType: 'dateTime',
        hideInSearch: true,
      },
      {
        title: intl.formatMessage({ id: 'binlog.list.endTime' }),
        dataIndex: 'endTime',
        width: 180,
        valueType: 'dateTime',
        hideInSearch: true,
      },
      {
        title: intl.formatMessage({ id: 'binlog.list.statusCol' }),
        dataIndex: 'status',
        width: 100,
        valueEnum: statusMap,
        render: (_, record) => {
          const status = statusMap[record.status] || {
            text: intl.formatMessage({ id: 'binlog.list.unknown' }),
            color: 'default',
          };
          return <Tag color={status.color}>{status.text}</Tag>;
        },
      },
      {
        title: intl.formatMessage({ id: 'binlog.list.createdAt' }),
        dataIndex: 'createdAt',
        width: 180,
        valueType: 'dateTime',
        hideInSearch: true,
      },
      {
        title: intl.formatMessage({ id: 'home.col.action' }),
        valueType: 'option',
        width: 120,
        fixed: 'right',
        render: (_, record) => (
          <Button
            type="link"
            icon={<EyeOutlined />}
            loading={detailLoading}
            onClick={() => loadDetail(record.id)}
          >
            {intl.formatMessage({ id: 'binlog.list.view' })}
          </Button>
        ),
      },
    ],
    [intl, statusMap, detailLoading, loadDetail]
  );

  const sqlColumns = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'binlog.list.seq' }),
        dataIndex: 'sequence',
        width: 60,
        render: (v: any) => v ?? '-',
      },
      {
        title: intl.formatMessage({ id: 'binlog.list.tableCol' }),
        dataIndex: 'tableName',
        width: 120,
      },
      {
        title: intl.formatMessage({ id: 'binlog.col.opType' }),
        dataIndex: 'operationType',
        width: 100,
        render: (v: string) => {
          const colorMap: Record<string, string> = {
            INSERT: 'green',
            UPDATE: 'blue',
            DELETE: 'red',
          };
          return <Tag color={colorMap[v] || 'default'}>{v}</Tag>;
        },
      },
      {
        title: intl.formatMessage({ id: 'binlog.col.rollbackSql' }),
        dataIndex: 'rollbackSql',
        ellipsis: true,
        render: (sql: string) =>
          sql ? (
            <Tooltip title={sql}>
              <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{sql}</span>
            </Tooltip>
          ) : (
            <span style={{ color: '#999' }}>-</span>
          ),
      },
    ],
    [intl]
  );

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pageTitle.binlog' })}
      extra={[
        <Button
          key="parse"
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setParseModalVisible(true)}
        >
          {intl.formatMessage({ id: 'binlog.createOrder' })}
        </Button>,
      ]}
    >
      <ProTable<API.BinlogOrder>
        columns={columns}
        request={async (params) => {
          try {
            const response = await listBinlogOrders({
              page: params.current || 1,
              pageSize: params.pageSize || 10,
              instanceId: params.instanceId,
            });
            const list = response.list || [];
            const total = response.pageResponse?.total || list.length;
            return {
              data: list,
              success: response.code === 200,
              total,
            };
          } catch {
            return { data: [], success: false, total: 0 };
          }
        }}
        rowKey="id"
        search={{ labelWidth: 'auto' }}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
        scroll={{ x: 1200 }}
      />

      <Modal
        title={intl.formatMessage({ id: 'binlog.modal.create' })}
        open={parseModalVisible}
        onCancel={() => {
          setParseModalVisible(false);
          form.resetFields();
          setSelectedInstanceId('');
          setSelectedDatabase('');
        }}
        footer={null}
        width={600}
      >
        <Form form={form} layout="vertical" onFinish={handleParse}>
          <Form.Item
            name="instanceId"
            label={intl.formatMessage({ id: 'queryEngine.col.instance' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'queryEngine.rule.instance' }) }]}
          >
            <Select
              placeholder={intl.formatMessage({ id: 'queryEngine.ph.instance' })}
              onChange={handleInstanceChange}
              showSearch
              filterOption={(input, option) =>
                String(option?.children ?? '').toLowerCase().includes(input.toLowerCase())
              }
            >
              {instances.map((inst: API.DatabaseInstance) => (
                <Option key={inst.id} value={inst.id}>
                  {inst.name} ({inst.host}:{inst.port})
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="databaseName"
            label={intl.formatMessage({ id: 'binlog.label.database' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'binlog.rule.selectDb' }) }]}
          >
            <Select
              placeholder={intl.formatMessage({ id: 'binlog.rule.selectDb' })}
              loading={databasesLoading}
              disabled={!selectedInstanceId}
              onChange={handleDatabaseChange}
              showSearch
              filterOption={(input, option) =>
                String(option?.children ?? '').toLowerCase().includes(input.toLowerCase())
              }
            >
              {databases.map((db: string) => (
                <Option key={db} value={db}>
                  {db}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="tableName"
            label={intl.formatMessage({ id: 'binlog.label.table' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'binlog.rule.selectTable' }) }]}
          >
            <Select
              placeholder={intl.formatMessage({ id: 'binlog.rule.selectTable' })}
              loading={tablesLoading}
              disabled={!selectedDatabase}
              showSearch
              filterOption={(input, option) =>
                String(option?.children ?? '').toLowerCase().includes(input.toLowerCase())
              }
            >
              {tables.map((table: string) => (
                <Option key={table} value={table}>
                  {table}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="timeRange"
            label={intl.formatMessage({ id: 'binlog.label.timeRange' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'binlog.rule.time' }) }]}
          >
            <RangePicker
              showTime
              format="YYYY-MM-DD HH:mm:ss"
              style={{ width: '100%' }}
            />
          </Form.Item>

          <Form.Item
            name="operationTypes"
            label={intl.formatMessage({ id: 'binlog.col.opType' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'binlog.rule.opTypes' }) }]}
          >
            <Select mode="multiple" placeholder={intl.formatMessage({ id: 'binlog.ph.opTypes' })}>
              <Option value="INSERT">INSERT</Option>
              <Option value="UPDATE">UPDATE</Option>
              <Option value="DELETE">DELETE</Option>
            </Select>
          </Form.Item>

          <Form.Item name="remark" label={intl.formatMessage({ id: 'binlog.label.remark' })}>
            <Select
              mode="tags"
              placeholder={intl.formatMessage({ id: 'binlog.remarkPh' })}
              maxTagCount={3}
              style={{ width: '100%' }}
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={parseLoading}>
                {intl.formatMessage({ id: 'binlog.submit' })}
              </Button>
              <Button
                onClick={() => {
                  setParseModalVisible(false);
                  form.resetFields();
                  setSelectedInstanceId('');
                  setSelectedDatabase('');
                }}
              >
                {intl.formatMessage({ id: 'binlog.cancel' })}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={intl.formatMessage({ id: 'binlog.detailTitle' })}
        open={detailModalVisible}
        onCancel={() => {
          setDetailModalVisible(false);
          setCurrentOrder(null);
          setCurrentOrderSqls([]);
        }}
        footer={null}
        width={1000}
        destroyOnClose
      >
        {currentOrder && (
          <div>
            <div style={{ marginBottom: 16 }}>
              <Space wrap size={[32, 8]}>
                <span>
                  <strong>{intl.formatMessage({ id: 'binlog.detail.workId' })}</strong>
                  {currentOrder.workId}
                </span>
                <span>
                  <strong>{intl.formatMessage({ id: 'binlog.detail.status' })}</strong>
                  <Tag color={statusMap[currentOrder.status]?.color}>
                    {statusMap[currentOrder.status]?.text}
                  </Tag>
                </span>
                <span>
                  <strong>{intl.formatMessage({ id: 'binlog.detail.instance' })}</strong>
                  {currentOrder.instanceId}
                </span>
                <span>
                  <strong>{intl.formatMessage({ id: 'binlog.detail.database' })}</strong>
                  {currentOrder.databaseName}
                </span>
                <span>
                  <strong>{intl.formatMessage({ id: 'binlog.detail.table' })}</strong>
                  {currentOrder.tableName}
                </span>
                <span>
                  <strong>{intl.formatMessage({ id: 'binlog.detail.time' })}</strong>
                  {currentOrder.startTime} ~ {currentOrder.endTime}
                </span>
                {currentOrder.remark && (
                  <span>
                    <strong>{intl.formatMessage({ id: 'binlog.detail.remark' })}</strong>
                    {currentOrder.remark}
                  </span>
                )}
              </Space>
            </div>

            <div>
              <strong>{intl.formatMessage({ id: 'binlog.detail.sqlList' })}</strong>
              <Table
                style={{ marginTop: 8 }}
                columns={sqlColumns}
                dataSource={currentOrderSqls}
                rowKey={(r, i) => r.id || String(i)}
                pagination={{ pageSize: 10 }}
                size="small"
                locale={{ emptyText: intl.formatMessage({ id: 'binlog.emptyRollback' }) }}
              />
            </div>
          </div>
        )}
      </Modal>
    </PageContainer>
  );
};
