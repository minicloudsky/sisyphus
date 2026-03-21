import { PageContainer, ProTable } from '@ant-design/pro-components';
import {
  Button,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Select,
  Space,
  Switch,
  Tag,
} from 'antd';
import { DeleteOutlined, EditOutlined, PlusOutlined } from '@ant-design/icons';
import {
  createArchiveConfig,
  deleteArchiveConfig,
  listArchiveConfigs,
  listDatabases,
  listInstances,
  listTables,
  updateArchiveConfig,
} from '@/services/api';
import type { ProColumns } from '@ant-design/pro-components';
import { useRequest } from 'ahooks';
import { useRef, useState, useMemo } from 'react';
import type { ActionType } from '@ant-design/pro-components';
import { useIntl } from '@/locales';

const { Option } = Select;

export default () => {
  const intl = useIntl();
  const actionRef = useRef<ActionType>();
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<API.ArchiveConfig | null>(null);
  const [createForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [sourceInstanceId, setSourceInstanceId] = useState<string>('');
  const [sourceDatabase, setSourceDatabase] = useState<string>('');
  const [destInstanceId, setDestInstanceId] = useState<string>('');
  const [destDatabase, setDestDatabase] = useState<string>('');
  const [submitting, setSubmitting] = useState(false);

  const { data: instancesData } = useRequest(() =>
    listInstances({ page: 1, pageSize: 1000, dbType: 'mysql' })
  );
  const instances = instancesData?.list || [];

  const { data: sourceDatabasesData, loading: sourceDatabasesLoading, run: loadSourceDatabases } =
    useRequest((instanceId: string) => listDatabases(instanceId), { manual: true });
  const sourceDatabases = sourceDatabasesData?.databases || [];

  const { data: sourceTablesData, loading: sourceTablesLoading, run: loadSourceTables } =
    useRequest(
      ({ instanceId, database }: { instanceId: string; database: string }) =>
        listTables(instanceId, database),
      { manual: true }
    );
  const sourceTables = (sourceTablesData as any)?.tables || [];

  const { data: destDatabasesData, loading: destDatabasesLoading, run: loadDestDatabases } =
    useRequest((instanceId: string) => listDatabases(instanceId), { manual: true });
  const destDatabases = destDatabasesData?.databases || [];

  const { data: destTablesData, loading: destTablesLoading, run: loadDestTables } =
    useRequest(
      ({ instanceId, database }: { instanceId: string; database: string }) =>
        listTables(instanceId, database),
      { manual: true }
    );
  const destTables = (destTablesData as any)?.tables || [];

  const handleSourceInstanceChange = (instanceId: string) => {
    setSourceInstanceId(instanceId);
    setSourceDatabase('');
    createForm.setFieldsValue({ sourceDatabase: undefined, sourceTable: undefined });
    if (instanceId) loadSourceDatabases(instanceId);
  };

  const handleSourceDatabaseChange = (database: string) => {
    setSourceDatabase(database);
    createForm.setFieldsValue({ sourceTable: undefined });
    if (sourceInstanceId && database)
      loadSourceTables({ instanceId: sourceInstanceId, database });
  };

  const handleDestInstanceChange = (instanceId: string) => {
    setDestInstanceId(instanceId);
    setDestDatabase('');
    createForm.setFieldsValue({ destDatabase: undefined, destTable: undefined });
    if (instanceId) loadDestDatabases(instanceId);
  };

  const handleDestDatabaseChange = (database: string) => {
    setDestDatabase(database);
    createForm.setFieldsValue({ destTable: undefined });
    if (destInstanceId && database)
      loadDestTables({ instanceId: destInstanceId, database });
  };

  const handleCreate = async () => {
    try {
      const values = await createForm.validateFields();
      setSubmitting(true);
      const response = await createArchiveConfig(values as API.CreateArchiveConfigRequest);
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'instance.createOk' }));
        setCreateModalVisible(false);
        createForm.resetFields();
        setSourceInstanceId('');
        setSourceDatabase('');
        setDestInstanceId('');
        setDestDatabase('');
        actionRef.current?.reload();
      } else {
        message.error(response.message || intl.formatMessage({ id: 'instance.createFail' }));
      }
    } catch (e: any) {
    } finally {
      setSubmitting(false);
    }
  };

  const handleEdit = async () => {
    if (!editingRecord) return;
    try {
      const values = await editForm.validateFields();
      setSubmitting(true);
      const response = await updateArchiveConfig(editingRecord.id, values);
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'common.updateOk' }));
        setEditModalVisible(false);
        setEditingRecord(null);
        actionRef.current?.reload();
      } else {
        message.error(response.message || intl.formatMessage({ id: 'common.operationFailed' }));
      }
    } catch (e: any) {
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    const response = await deleteArchiveConfig(id);
    if (response.code === 200) {
      message.success(intl.formatMessage({ id: 'common.deleteOk' }));
      actionRef.current?.reload();
    } else {
      message.error(response.message || intl.formatMessage({ id: 'common.deleteFail' }));
    }
  };

  const columns: ProColumns<API.ArchiveConfig>[] = useMemo(
    () => [
      { title: intl.formatMessage({ id: 'archive.col.taskName' }), dataIndex: 'taskName', width: 150 },
      { title: intl.formatMessage({ id: 'common.id' }), dataIndex: 'id', width: 80 },
      {
        title: intl.formatMessage({ id: 'archive.col.sourceTable' }),
        dataIndex: 'sourceTable',
        width: 200,
        render: (_, record) => `${record.sourceDatabase}.${record.sourceTable}`,
      },
      {
        title: intl.formatMessage({ id: 'archive.col.destTable' }),
        dataIndex: 'destTable',
        width: 200,
        render: (_, record) =>
          record.destInstanceId
            ? `${record.destDatabase}.${record.destTable}`
            : intl.formatMessage({ id: 'archive.modeDelete' }),
      },
      { title: intl.formatMessage({ id: 'archive.col.archiveField' }), dataIndex: 'archiveField', width: 120 },
      { title: intl.formatMessage({ id: 'archive.col.retentionDays' }), dataIndex: 'retentionDays', width: 100 },
      { title: intl.formatMessage({ id: 'archive.col.cron' }), dataIndex: 'cronExpression', width: 160 },
      {
        title: intl.formatMessage({ id: 'instance.col.status' }),
        dataIndex: 'isEnabled',
        width: 80,
        render: (_, record) => (
          <Tag color={record.isEnabled ? 'success' : 'default'}>
            {record.isEnabled
              ? intl.formatMessage({ id: 'common.enabled' })
              : intl.formatMessage({ id: 'common.disabled' })}
          </Tag>
        ),
      },
      { title: intl.formatMessage({ id: 'common.createdAt' }), dataIndex: 'createdAt', width: 180, hideInSearch: true },
      {
        title: intl.formatMessage({ id: 'home.col.action' }),
        valueType: 'option',
        width: 150,
        render: (_, record) => (
          <Space>
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => {
                setEditingRecord(record);
                editForm.setFieldsValue(record);
                setEditModalVisible(true);
              }}
            >
              {intl.formatMessage({ id: 'common.edit' })}
            </Button>
            <Popconfirm title={intl.formatMessage({ id: 'archive.confirmDelete' })} onConfirm={() => handleDelete(record.id)}>
              <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                {intl.formatMessage({ id: 'common.delete' })}
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [intl]
  );

  const instanceOptions = instances.map((inst: any) => ({
    label: `${inst.name} (${inst.host}:${inst.port})`,
    value: inst.id,
  }));

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'archive.pageTitle' })}
      extra={[
        <Button
          key="create"
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            createForm.resetFields();
            setSourceInstanceId('');
            setSourceDatabase('');
            setDestInstanceId('');
            setDestDatabase('');
            setCreateModalVisible(true);
          }}
        >
          {intl.formatMessage({ id: 'archive.createConfig' })}
        </Button>,
      ]}
    >
      <ProTable<API.ArchiveConfig>
        actionRef={actionRef}
        columns={columns}
        request={async (params) => {
          const response = await listArchiveConfigs({
            page: params.current || 1,
            pageSize: params.pageSize || 10,
          });
          return {
            data: response.list || [],
            success: response.code === 200,
            total: response.pageResponse?.total || 0,
          };
        }}
        rowKey="id"
        search={false}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
        scroll={{ x: 1400 }}
      />

      <Modal
        title={intl.formatMessage({ id: 'archive.createModal' })}
        open={createModalVisible}
        onOk={handleCreate}
        onCancel={() => setCreateModalVisible(false)}
        okText={intl.formatMessage({ id: 'archive.createBtn' })}
        cancelText={intl.formatMessage({ id: 'common.cancel' })}
        confirmLoading={submitting}
        width={800}
        destroyOnClose
      >
        <Form form={createForm} layout="vertical">
          <Form.Item name="taskName" label={intl.formatMessage({ id: 'archive.col.taskName' })} rules={[{ required: true, message: intl.formatMessage({ id: 'archive.rule.taskName' }) }]}>
            <Input placeholder={intl.formatMessage({ id: 'archive.ph.taskName' })} />
          </Form.Item>
          <Form.Item name="instanceId" label={intl.formatMessage({ id: 'archive.label.sourceInst' })} rules={[{ required: true, message: intl.formatMessage({ id: 'archive.rule.sourceInst' }) }]}>
            <Select
              placeholder={intl.formatMessage({ id: 'archive.rule.sourceInst' })}
              showSearch
              filterOption={(input, option) =>
                (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
              }
              options={instanceOptions}
              onChange={handleSourceInstanceChange}
            />
          </Form.Item>
          <Form.Item name="sourceDatabase" label={intl.formatMessage({ id: 'archive.label.sourceDb' })} rules={[{ required: true, message: intl.formatMessage({ id: 'archive.rule.sourceDb' }) }]}>
            <Select
              placeholder={intl.formatMessage({ id: 'archive.rule.sourceDb' })}
              loading={sourceDatabasesLoading}
              disabled={!sourceInstanceId}
              onChange={handleSourceDatabaseChange}
              showSearch
            >
              {sourceDatabases.map((db: string) => (
                <Option key={db} value={db}>{db}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="sourceTable" label={intl.formatMessage({ id: 'archive.label.sourceTable' })} rules={[{ required: true, message: intl.formatMessage({ id: 'archive.rule.sourceTable' }) }]}>
            <Select
              placeholder={intl.formatMessage({ id: 'archive.rule.sourceTable' })}
              loading={sourceTablesLoading}
              disabled={!sourceDatabase}
              showSearch
            >
              {sourceTables.map((t: string) => (
                <Option key={t} value={t}>{t}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="destInstanceId" label={intl.formatMessage({ id: 'archive.label.destInst' })}>
            <Select
              placeholder={intl.formatMessage({ id: 'archive.ph.destInst' })}
              allowClear
              showSearch
              filterOption={(input, option) =>
                (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
              }
              options={instanceOptions}
              onChange={handleDestInstanceChange}
            />
          </Form.Item>
          {destInstanceId && (
            <>
              <Form.Item name="destDatabase" label={intl.formatMessage({ id: 'archive.label.destDb' })} rules={[{ required: true, message: intl.formatMessage({ id: 'archive.rule.destDb' }) }]}>
                <Select
                  placeholder={intl.formatMessage({ id: 'archive.rule.destDb' })}
                  loading={destDatabasesLoading}
                  onChange={handleDestDatabaseChange}
                  showSearch
                >
                  {destDatabases.map((db: string) => (
                    <Option key={db} value={db}>{db}</Option>
                  ))}
                </Select>
              </Form.Item>
              <Form.Item name="destTable" label={intl.formatMessage({ id: 'archive.label.destTable' })} rules={[{ required: true, message: intl.formatMessage({ id: 'archive.rule.destTable' }) }]}>
                <Select
                  placeholder={intl.formatMessage({ id: 'archive.rule.destTable' })}
                  loading={destTablesLoading}
                  disabled={!destDatabase}
                  showSearch
                >
                  {destTables.map((t: string) => (
                    <Option key={t} value={t}>{t}</Option>
                  ))}
                </Select>
              </Form.Item>
            </>
          )}
          <Form.Item name="archiveField" label={intl.formatMessage({ id: 'archive.label.archiveField' })} rules={[{ required: true, message: intl.formatMessage({ id: 'archive.rule.archiveField' }) }]} extra={intl.formatMessage({ id: 'archive.extra.archiveField' })}>
            <Input placeholder={intl.formatMessage({ id: 'archive.ph.archiveField' })} />
          </Form.Item>
          <Form.Item name="retentionDays" label={intl.formatMessage({ id: 'archive.label.retentionDays' })} rules={[{ required: true, message: intl.formatMessage({ id: 'archive.rule.retentionDays' }) }]}>
            <InputNumber min={1} style={{ width: '100%' }} placeholder={intl.formatMessage({ id: 'archive.ph.retentionDays' })} />
          </Form.Item>
          <Form.Item name="archiveCondition" label={intl.formatMessage({ id: 'archive.label.condition' })} extra={intl.formatMessage({ id: 'archive.extra.condition' })}>
            <Input placeholder={intl.formatMessage({ id: 'archive.ph.condition' })} />
          </Form.Item>
          <Form.Item name="cronExpression" label={intl.formatMessage({ id: 'archive.label.cron' })} rules={[{ required: true, message: intl.formatMessage({ id: 'archive.rule.cron' }) }]} extra={intl.formatMessage({ id: 'archive.extra.cron' })}>
            <Input placeholder="0 2 * * *" />
          </Form.Item>
          <Form.Item name="isEnabled" label={intl.formatMessage({ id: 'archive.label.enabled' })} valuePropName="checked" initialValue={true}>
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={intl.formatMessage({ id: 'archive.editModal' })}
        open={editModalVisible}
        onOk={handleEdit}
        onCancel={() => { setEditModalVisible(false); setEditingRecord(null); }}
        okText={intl.formatMessage({ id: 'common.save' })}
        cancelText={intl.formatMessage({ id: 'common.cancel' })}
        confirmLoading={submitting}
        width={600}
        destroyOnClose
      >
        <Form form={editForm} layout="vertical">
          <Form.Item name="taskName" label={intl.formatMessage({ id: 'archive.col.taskName' })} rules={[{ required: true, message: intl.formatMessage({ id: 'archive.rule.taskName' }) }]}>
            <Input />
          </Form.Item>
          <Form.Item name="archiveCondition" label={intl.formatMessage({ id: 'archive.label.condition' })}>
            <Input />
          </Form.Item>
          <Form.Item name="cronExpression" label={intl.formatMessage({ id: 'archive.label.cron' })} rules={[{ required: true, message: intl.formatMessage({ id: 'archive.rule.cron' }) }]}>
            <Input />
          </Form.Item>
          <Form.Item name="retentionDays" label={intl.formatMessage({ id: 'archive.label.retentionDays' })}>
            <InputNumber min={1} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="isEnabled" label={intl.formatMessage({ id: 'archive.label.enabled' })} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
};
