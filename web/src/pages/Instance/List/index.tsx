import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Button, Space, Tag, message, Popconfirm, Modal, Form, Input, Select, Switch, InputNumber } from 'antd';
import DbTypeIcon from '@/components/DbTypeIcon';
import { PlusOutlined, EditOutlined, DeleteOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { listInstances, deleteInstance, testConnection, getInstance, updateInstance, createInstance } from '@/services/api';
import type { ProColumns, ActionType } from '@ant-design/pro-components';
import { useRequest } from 'ahooks';
import { useState, useRef, useMemo } from 'react';
import { formatDateTime } from '@/utils/format';
import { useIntl } from '@/locales';

const { Option } = Select;
const { TextArea } = Input;

export default () => {
  const intl = useIntl();
  const actionRef = useRef<ActionType>();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [editingInstance, setEditingInstance] = useState<API.DatabaseInstance | null>(null);
  const [form] = Form.useForm();
  const [createForm] = Form.useForm();
  const { run: handleDelete } = useRequest(
    async (id: string) => {
      const response = await deleteInstance(id);
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'common.deleteOk' }));
        return true;
      } else {
        message.error(response.message || intl.formatMessage({ id: 'common.deleteFail' }));
        return false;
      }
    },
    { manual: true }
  );

  const { run: handleTest } = useRequest(
    async (id: string) => {
      const response = await testConnection(id);
      if (response.code === 200 && response.data) {
        if (response.data.success) {
          message.success(intl.formatMessage({ id: 'common.testConnOk' }));
        } else {
          message.error(response.data.message || intl.formatMessage({ id: 'common.testConnFail' }));
        }
      } else {
        message.error(response.message || intl.formatMessage({ id: 'common.testConnFail' }));
      }
    },
    { manual: true }
  );

  const handleEdit = async (record: API.DatabaseInstance) => {
    try {
      const response = await getInstance(record.id);
      if (response.code === 200 && response.data) {
        setEditingInstance(response.data);
        form.setFieldsValue({
          name: response.data.name,
          host: response.data.host,
          port: response.data.port,
          username: response.data.username,
          env: response.data.env,
          description: response.data.description,
          isEnabled: response.data.status === 1,
        });
        setEditModalVisible(true);
      } else {
        message.error(response.message || intl.formatMessage({ id: 'instance.fetchFail' }));
      }
    } catch {
    }
  };

  const { loading: saving, run: handleSave } = useRequest(
    async () => {
      const values = await form.validateFields();
      if (!editingInstance) return;

      const updateData: API.UpdateInstanceRequest = {
        name: values.name,
        host: values.host,
        port: values.port,
        username: values.username,
        env: values.env,
        description: values.description,
        status: values.isEnabled ? 1 : 0,
      };

      if (values.password) {
        updateData.password = values.password;
      }

      const response = await updateInstance(editingInstance.id, updateData);

      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'common.updateOk' }));
        setEditModalVisible(false);
        form.resetFields();
        setEditingInstance(null);
        actionRef.current?.reload();
      } else {
        message.error(response.message || intl.formatMessage({ id: 'common.operationFailed' }));
      }
    },
    { manual: true }
  );

  const handleCancelEdit = () => {
    setEditModalVisible(false);
    form.resetFields();
    setEditingInstance(null);
  };

  const handleCreate = () => {
    createForm.resetFields();
    createForm.setFieldsValue({
      env: 'dev',
      dbType: 'mysql',
    });
    setCreateModalVisible(true);
  };

  const { loading: creating, run: handleCreateSave } = useRequest(
    async () => {
      const values = await createForm.validateFields();

      const response = await createInstance({
        name: values.name,
        connectionName: values.connectionName || values.name,
        dbType: values.dbType,
        host: values.host,
        port: values.port,
        username: values.username,
        password: values.password,
        env: values.env,
        description: values.description,
      });

      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'instance.createOk' }));
        setCreateModalVisible(false);
        createForm.resetFields();
        actionRef.current?.reload();
      } else {
        message.error(response.message || intl.formatMessage({ id: 'instance.createFail' }));
      }
    },
    { manual: true }
  );

  const handleCancelCreate = () => {
    setCreateModalVisible(false);
    createForm.resetFields();
  };

  const columns: ProColumns<API.DatabaseInstance>[] = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'instance.col.name' }),
        dataIndex: 'name',
        width: 150,
        fixed: 'left',
        render: (text) => text || '-',
      },
      {
        title: intl.formatMessage({ id: 'instance.col.connName' }),
        dataIndex: 'connectionName',
        width: 120,
        render: (text) => text || '-',
      },
      {
        title: intl.formatMessage({ id: 'instance.col.dbType' }),
        dataIndex: 'dbType',
        width: 100,
        valueEnum: {
          mysql: { text: 'MySQL' },
          redis: { text: 'Redis' },
          clickhouse: { text: 'ClickHouse' },
          hbase: { text: 'HBase' },
        },
        render: (_, record) => {
          const labelMap: Record<string, string> = {
            mysql: 'MySQL', redis: 'Redis', clickhouse: 'ClickHouse', hbase: 'HBase',
          };
          return (
            <Space size={6}>
              <DbTypeIcon dbType={record.dbType} size={18} />
              <span style={{ fontSize: 13 }}>{labelMap[record.dbType] || record.dbType}</span>
            </Space>
          );
        },
      },
      {
        title: intl.formatMessage({ id: 'instance.col.connInfo' }),
        dataIndex: 'host',
        width: 200,
        render: (_, record) => `${record.host}:${record.port}`,
      },
      {
        title: intl.formatMessage({ id: 'instance.col.username' }),
        dataIndex: 'username',
        width: 120,
      },
      {
        title: intl.formatMessage({ id: 'instance.col.env' }),
        dataIndex: 'env',
        width: 80,
        valueEnum: {
          dev: { text: intl.formatMessage({ id: 'env.dev' }) },
          test: { text: intl.formatMessage({ id: 'env.test' }) },
          prod: { text: intl.formatMessage({ id: 'env.prod' }) },
        },
        render: (_, record) => {
          const envMap: Record<string, { text: string; color: string }> = {
            dev: { text: intl.formatMessage({ id: 'env.dev' }), color: 'green' },
            test: { text: intl.formatMessage({ id: 'env.test' }), color: 'orange' },
            prod: { text: intl.formatMessage({ id: 'env.prod' }), color: 'red' },
          };
          const env = envMap[record.env] || { text: record.env, color: 'default' };
          return <Tag color={env.color}>{env.text}</Tag>;
        },
      },
      {
        title: intl.formatMessage({ id: 'instance.col.status' }),
        dataIndex: 'status',
        width: 80,
        valueEnum: {
          1: { text: intl.formatMessage({ id: 'user.status.normal' }) },
          0: { text: intl.formatMessage({ id: 'user.status.disabled' }) },
        },
        render: (_, record) => (
          <Tag color={record.status === 1 ? 'success' : 'default'}>
            {record.status === 1
              ? intl.formatMessage({ id: 'user.status.normal' })
              : intl.formatMessage({ id: 'user.status.disabled' })}
          </Tag>
        ),
      },
      {
        title: intl.formatMessage({ id: 'instance.col.owner' }),
        dataIndex: 'owner',
        width: 100,
      },
      {
        title: intl.formatMessage({ id: 'common.createdAt' }),
        dataIndex: 'createdAt',
        width: 180,
        hideInSearch: true,
        render: (_: any, record: any) => formatDateTime(record.createdAt),
      },
      {
        title: intl.formatMessage({ id: 'home.col.action' }),
        valueType: 'option',
        width: 200,
        fixed: 'right',
        render: (_, record) => (
          <Space>
            <Button
              type="link"
              icon={<CheckCircleOutlined />}
              onClick={() => handleTest(record.id)}
            >
              {intl.formatMessage({ id: 'common.test' })}
            </Button>
            <Button
              type="link"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
            >
              {intl.formatMessage({ id: 'common.edit' })}
            </Button>
            <Popconfirm
              title={intl.formatMessage({ id: 'instance.confirmDelete' })}
              onConfirm={() => handleDelete(record.id)}
            >
              <Button type="link" danger icon={<DeleteOutlined />}>
                {intl.formatMessage({ id: 'common.delete' })}
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [intl]
  );

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'instance.title' })}
      extra={[
        <Button
          key="create"
          type="primary"
          icon={<PlusOutlined />}
          onClick={handleCreate}
        >
          {intl.formatMessage({ id: 'instance.add' })}
        </Button>,
      ]}
    >
      <ProTable<API.DatabaseInstance>
        actionRef={actionRef}
        columns={columns}
        request={async (params) => {
          const response = await listInstances({
            page: params.current || 1,
            pageSize: params.pageSize || 10,
            dbType: params.dbType,
            env: params.env,
          });
          return {
            data: response.list || [],
            success: response.code === 200,
            total: response.pageResponse?.total || 0,
          };
        }}
        rowKey="id"
        search={{
          labelWidth: 'auto',
        }}
        pagination={{
          defaultPageSize: 10,
          showSizeChanger: true,
        }}
        scroll={{ x: 1400 }}
      />

      <Modal
        title={intl.formatMessage({ id: 'instance.editTitle' })}
        open={editModalVisible}
        onOk={handleSave}
        onCancel={handleCancelEdit}
        confirmLoading={saving}
        width={600}
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            isEnabled: true,
          }}
        >
          <Form.Item
            name="name"
            label={intl.formatMessage({ id: 'instance.col.name' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'instance.nameRule' }) }]}
          >
            <Input placeholder={intl.formatMessage({ id: 'instance.nameRule' })} />
          </Form.Item>

          <Form.Item
            name="host"
            label={intl.formatMessage({ id: 'instance.host' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'instance.hostRule' }) }]}
          >
            <Input placeholder={intl.formatMessage({ id: 'instance.hostRule' })} />
          </Form.Item>

          <Form.Item
            name="port"
            label={intl.formatMessage({ id: 'instance.port' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'instance.portRule' }) }]}
          >
            <InputNumber min={1} max={65535} style={{ width: '100%' }} placeholder={intl.formatMessage({ id: 'instance.portRule' })} />
          </Form.Item>

          <Form.Item
            name="username"
            label={intl.formatMessage({ id: 'instance.col.username' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'instance.userRule' }) }]}
          >
            <Input placeholder={intl.formatMessage({ id: 'instance.userRule' })} />
          </Form.Item>

          <Form.Item
            name="password"
            label={intl.formatMessage({ id: 'login.password' })}
            tooltip={intl.formatMessage({ id: 'instance.pwdTooltip' })}
          >
            <Input.Password placeholder={intl.formatMessage({ id: 'instance.pwdPh' })} />
          </Form.Item>

          <Form.Item
            name="env"
            label={intl.formatMessage({ id: 'instance.col.env' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'instance.envRule' }) }]}
          >
            <Select placeholder={intl.formatMessage({ id: 'instance.envRule' })}>
              <Option value="dev">{intl.formatMessage({ id: 'env.dev' })}</Option>
              <Option value="test">{intl.formatMessage({ id: 'env.test' })}</Option>
              <Option value="prod">{intl.formatMessage({ id: 'env.prod' })}</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="isEnabled"
            label={intl.formatMessage({ id: 'instance.col.status' })}
            valuePropName="checked"
          >
            <Switch checkedChildren={intl.formatMessage({ id: 'instance.switchOn' })} unCheckedChildren={intl.formatMessage({ id: 'instance.switchOff' })} />
          </Form.Item>

          <Form.Item
            name="description"
            label={intl.formatMessage({ id: 'common.description' })}
          >
            <TextArea rows={3} placeholder={intl.formatMessage({ id: 'instance.descPh' })} maxLength={500} showCount />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={intl.formatMessage({ id: 'instance.createTitle' })}
        open={createModalVisible}
        onOk={handleCreateSave}
        onCancel={handleCancelCreate}
        confirmLoading={creating}
        width={600}
      >
        <Form
          form={createForm}
          layout="vertical"
          initialValues={{
            env: 'dev',
            dbType: 'mysql',
          }}
        >
          <Form.Item
            name="name"
            label={intl.formatMessage({ id: 'instance.col.name' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'instance.nameRule' }) }]}
          >
            <Input placeholder={intl.formatMessage({ id: 'instance.nameRule' })} />
          </Form.Item>

          <Form.Item
            name="connectionName"
            label={intl.formatMessage({ id: 'instance.col.connName' })}
            tooltip={intl.formatMessage({ id: 'instance.connNameTooltip' })}
          >
            <Input placeholder={intl.formatMessage({ id: 'instance.connNamePh' })} />
          </Form.Item>

          <Form.Item
            name="dbType"
            label={intl.formatMessage({ id: 'instance.col.dbType' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'instance.dbTypeRule' }) }]}
          >
            <Select placeholder={intl.formatMessage({ id: 'instance.dbTypeRule' })} optionLabelProp="label">
              <Option value="mysql" label="MySQL">
                <Space size={8}>
                  <DbTypeIcon dbType="mysql" size={18} />
                  MySQL
                </Space>
              </Option>
              <Option value="redis" label="Redis">
                <Space size={8}>
                  <DbTypeIcon dbType="redis" size={18} />
                  Redis
                </Space>
              </Option>
              <Option value="clickhouse" label="ClickHouse">
                <Space size={8}>
                  <DbTypeIcon dbType="clickhouse" size={18} />
                  ClickHouse
                </Space>
              </Option>
              <Option value="hbase" label="HBase">
                <Space size={8}>
                  <DbTypeIcon dbType="hbase" size={18} />
                  HBase
                </Space>
              </Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="host"
            label={intl.formatMessage({ id: 'instance.host' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'instance.hostRule' }) }]}
          >
            <Input placeholder={intl.formatMessage({ id: 'instance.hostRule' })} />
          </Form.Item>

          <Form.Item
            name="port"
            label={intl.formatMessage({ id: 'instance.port' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'instance.portRule' }) }]}
          >
            <InputNumber min={1} max={65535} style={{ width: '100%' }} placeholder={intl.formatMessage({ id: 'instance.portRule' })} />
          </Form.Item>

          <Form.Item
            name="username"
            label={intl.formatMessage({ id: 'instance.col.username' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'instance.userRule' }) }]}
          >
            <Input placeholder={intl.formatMessage({ id: 'instance.userRule' })} />
          </Form.Item>

          <Form.Item
            name="password"
            label={intl.formatMessage({ id: 'login.password' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'instance.pwdCreateRule' }) }]}
          >
            <Input.Password placeholder={intl.formatMessage({ id: 'instance.pwdCreateRule' })} />
          </Form.Item>

          <Form.Item
            name="env"
            label={intl.formatMessage({ id: 'instance.col.env' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'instance.envRule' }) }]}
          >
            <Select placeholder={intl.formatMessage({ id: 'instance.envRule' })}>
              <Option value="dev">{intl.formatMessage({ id: 'env.dev' })}</Option>
              <Option value="test">{intl.formatMessage({ id: 'env.test' })}</Option>
              <Option value="prod">{intl.formatMessage({ id: 'env.prod' })}</Option>
            </Select>
          </Form.Item>

          <Form.Item
            name="description"
            label={intl.formatMessage({ id: 'common.description' })}
          >
            <TextArea rows={3} placeholder={intl.formatMessage({ id: 'instance.descPh' })} maxLength={500} showCount />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  );
};
