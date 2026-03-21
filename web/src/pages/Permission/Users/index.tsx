import { PageContainer, ProTable, ModalForm, ProFormText, ProFormSelect, ProFormSwitch } from '@ant-design/pro-components';
import { Button, Space, Tag, message, Popconfirm } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, CloudServerOutlined } from '@ant-design/icons';
import { listUsers, createUser, updateUser, deleteUser } from '@/services/api';
import { InstancePermissionsDrawer } from '../components/InstancePermissionsDrawer';
import type { ProColumns } from '@ant-design/pro-components';
import { useRequest } from 'ahooks';
import { useState, useMemo } from 'react';
import { useIntl } from '@/locales';

export default () => {
  const intl = useIntl();
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<API.User | null>(null);
  const [instDrawerOpen, setInstDrawerOpen] = useState(false);
  const [instDrawerUser, setInstDrawerUser] = useState<API.User | null>(null);

  const { run: handleDelete } = useRequest(
    async (id: string) => {
      const response = await deleteUser(id);
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

  const columns: ProColumns<API.User>[] = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'common.id' }),
        dataIndex: 'id',
        width: 100,
      },
      {
        title: intl.formatMessage({ id: 'login.username' }),
        dataIndex: 'username',
        width: 120,
      },
      {
        title: intl.formatMessage({ id: 'register.email' }),
        dataIndex: 'email',
        width: 180,
      },
      {
        title: intl.formatMessage({ id: 'register.realName' }),
        dataIndex: 'realName',
        width: 120,
      },
      {
        title: intl.formatMessage({ id: 'user.col.phone' }),
        dataIndex: 'phone',
        width: 120,
      },
      {
        title: intl.formatMessage({ id: 'user.col.department' }),
        dataIndex: 'department',
        width: 120,
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
        title: intl.formatMessage({ id: 'home.col.action' }),
        valueType: 'option',
        width: 220,
        render: (_, record) => (
          <Space>
            <Button
              type="link"
              icon={<CloudServerOutlined />}
              onClick={() => {
                setInstDrawerUser(record);
                setInstDrawerOpen(true);
              }}
            >
              {intl.formatMessage({ id: 'user.instancePerm' })}
            </Button>
            <Button
              type="link"
              icon={<EditOutlined />}
              onClick={() => {
                setEditingRecord(record);
                setEditModalVisible(true);
              }}
            >
              {intl.formatMessage({ id: 'common.edit' })}
            </Button>
            <Popconfirm
              title={intl.formatMessage({ id: 'user.confirmDelete' })}
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
      title={intl.formatMessage({ id: 'pageTitle.permissionUsers' })}
      extra={[
        <Button
          key="create"
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setCreateModalVisible(true)}
        >
          {intl.formatMessage({ id: 'user.addUser' })}
        </Button>,
      ]}
    >
      <ProTable<API.User>
        columns={columns}
        request={async (params) => {
          const response = await listUsers({
            page: params.current || 1,
            pageSize: params.pageSize || 10,
          });
          // 自定义路由 writeSuccess：列表在 data 里，不是顶层 list / pageResponse
          const res = response as API.ListUsersResponse & {
            data?: { list?: API.User[]; total?: number };
          };
          const payload = res.data ?? (res as { list?: API.User[]; total?: number });
          const raw = payload.list || [];
          const data = raw.map((u: API.User & { real_name?: string }) => ({
            ...u,
            realName: u.realName ?? u.real_name ?? '',
          }));
          return {
            data,
            success: response.code === 200,
            total: payload.total ?? 0,
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
        scroll={{ x: 1200 }}
      />

      <ModalForm
        title={intl.formatMessage({ id: 'user.addUser' })}
        open={createModalVisible}
        onOpenChange={setCreateModalVisible}
        onFinish={async (values) => {
          const response = await createUser(values as API.CreateUserRequest);
          if (response.code === 200) {
            message.success(intl.formatMessage({ id: 'user.addOk' }));
            setCreateModalVisible(false);
            return true;
          } else {
            message.error(response.message || intl.formatMessage({ id: 'user.addFail' }));
            return false;
          }
        }}
        width={600}
      >
        <ProFormText
          name="username"
          label={intl.formatMessage({ id: 'login.username' })}
          rules={[{ required: true, message: intl.formatMessage({ id: 'login.usernameRequired' }) }]}
        />
        <ProFormText.Password
          name="password"
          label={intl.formatMessage({ id: 'login.password' })}
          rules={[
            { required: true, message: intl.formatMessage({ id: 'login.passwordRequired' }) },
            { min: 6, message: intl.formatMessage({ id: 'register.passwordMin' }) },
          ]}
        />
        <ProFormText
          name="email"
          label={intl.formatMessage({ id: 'register.email' })}
          rules={[
            { required: true, message: intl.formatMessage({ id: 'register.emailRequired' }) },
            { type: 'email', message: intl.formatMessage({ id: 'register.emailInvalid' }) },
          ]}
        />
        <ProFormText
          name="realName"
          label={intl.formatMessage({ id: 'register.realName' })}
          rules={[{ required: true, message: intl.formatMessage({ id: 'register.realNameRequired' }) }]}
        />
        <ProFormText name="phone" label={intl.formatMessage({ id: 'user.col.phone' })} />
        <ProFormText name="department" label={intl.formatMessage({ id: 'user.col.department' })} />
      </ModalForm>

      <ModalForm
        title={intl.formatMessage({ id: 'user.editUser' })}
        open={editModalVisible}
        onOpenChange={setEditModalVisible}
        initialValues={editingRecord || {}}
        onFinish={async (values) => {
          if (!editingRecord) return false;
          const response = await updateUser(editingRecord.id, values);
          if (response.code === 200) {
            message.success(intl.formatMessage({ id: 'common.updateOk' }));
            setEditModalVisible(false);
            setEditingRecord(null);
            return true;
          } else {
            message.error(response.message || intl.formatMessage({ id: 'common.operationFailed' }));
            return false;
          }
        }}
        width={600}
      >
        <ProFormText
          name="email"
          label={intl.formatMessage({ id: 'register.email' })}
          rules={[{ type: 'email', message: intl.formatMessage({ id: 'register.emailInvalid' }) }]}
        />
        <ProFormText name="realName" label={intl.formatMessage({ id: 'register.realName' })} />
        <ProFormText name="phone" label={intl.formatMessage({ id: 'user.col.phone' })} />
        <ProFormText name="department" label={intl.formatMessage({ id: 'user.col.department' })} />
        <ProFormSwitch name="status" label={intl.formatMessage({ id: 'instance.col.status' })} />
      </ModalForm>

      <InstancePermissionsDrawer
        open={instDrawerOpen}
        onClose={() => {
          setInstDrawerOpen(false);
          setInstDrawerUser(null);
        }}
        mode="user"
        targetId={instDrawerUser?.id ? String(instDrawerUser.id) : ''}
        targetLabel={instDrawerUser?.username}
      />
    </PageContainer>
  );
};
