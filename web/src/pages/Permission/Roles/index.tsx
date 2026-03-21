import { PageContainer, ProTable, ModalForm, ProFormText, ProFormTextArea } from '@ant-design/pro-components';
import { Button, Space, message, Drawer, Tree, Spin, Tag, Typography, Dropdown, Modal } from 'antd';
import type { MenuProps } from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SafetyOutlined,
  EllipsisOutlined,
  CloudServerOutlined,
} from '@ant-design/icons';
import { listRoles, createRole, updateRole, deleteRole, getRolePermissions, assignPermissionsToRole, listPermissions } from '@/services/api';
import type { ProColumns } from '@ant-design/pro-components';
import { useRequest } from 'ahooks';
import { useState, useRef, useMemo, type ReactNode } from 'react';
import { useIntl } from '@/locales';
import { SYSTEM_PERMISSIONS, PERM_MODULE_ORDER, type PermModuleKey } from '../permissionsCatalog';
import { InstancePermissionsDrawer } from '../components/InstancePermissionsDrawer';

const { Text } = Typography;

const ACTION_COLOR: Record<string, string> = {
  view: 'blue', create: 'green', update: 'orange', delete: 'red',
  execute: 'purple', approve: 'gold', export: 'cyan', manage: 'geekblue',
};

function buildTreeData(backendPerms: API.Permission[], intl: ReturnType<typeof useIntl>) {
  const codeToId = new Map(backendPerms.map((p) => [p.code, String(p.id)]));
  return PERM_MODULE_ORDER.map((mod: PermModuleKey) => {
    const children = SYSTEM_PERMISSIONS.filter((p) => p.module === mod).map((p) => ({
      key: codeToId.get(p.code) || `__missing__${p.code}`,
      title: (
        <Space size={4}>
          <span>{intl.formatMessage({ id: p.nameKey })}</span>
          <Tag color={ACTION_COLOR[p.action] || 'default'} style={{ fontSize: 11, lineHeight: '16px', padding: '0 4px' }}>
            {intl.formatMessage({ id: `perm.action.${p.action}` })}
          </Tag>
          <Text type="secondary" style={{ fontSize: 11 }}>{p.code}</Text>
        </Space>
      ),
      disabled: !codeToId.has(p.code),
    }));
    if (!children.length) return null;
    return {
      key: `module:${mod}`,
      title: intl.formatMessage({ id: `perm.module.${mod}` }),
      children,
    };
  }).filter(Boolean) as { key: string; title: ReactNode; children: any[] }[];
}

export default () => {
  const intl = useIntl();
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<API.Role | null>(null);
  const tableRef = useRef<any>(null);

  // Permission drawer state
  const [permDrawerVisible, setPermDrawerVisible] = useState(false);
  const [permRole, setPermRole] = useState<API.Role | null>(null);
  const [treeData, setTreeData] = useState<any[]>([]);
  const [checkedKeys, setCheckedKeys] = useState<string[]>([]);
  const [loadingPerms, setLoadingPerms] = useState(false);
  const [savingPerms, setSavingPerms] = useState(false);
  const [instDrawerVisible, setInstDrawerVisible] = useState(false);
  const [instRole, setInstRole] = useState<API.Role | null>(null);

  const openPermDrawer = async (role: API.Role) => {
    setPermRole(role);
    setPermDrawerVisible(true);
    setLoadingPerms(true);
    try {
      // Load all backend permissions and current role assignments in parallel
      const [allPermsRes, rolePermsRes] = await Promise.all([
        listPermissions({ page: 1, pageSize: 200 }),
        getRolePermissions(role.id),
      ]);
      const allPerms: API.Permission[] = allPermsRes.list || [];
      const rolePermIds = new Set(
        ((rolePermsRes as any).list || []).map((p: any) => String(p.id))
      );
      setTreeData(buildTreeData(allPerms, intl));
      setCheckedKeys(Array.from(rolePermIds) as string[]);
    } catch {
    } finally {
      setLoadingPerms(false);
    }
  };

  const handleSavePerms = async () => {
    if (!permRole) return;
    setSavingPerms(true);
    try {
      // Filter out module-level keys and missing keys
      const ids = checkedKeys
        .filter((k) => !k.startsWith('module:') && !k.startsWith('__missing__'));
      const res = await assignPermissionsToRole(permRole.id, ids);
      if ((res as any).code === 200) {
        message.success(intl.formatMessage({ id: 'role.permSaved' }));
        setPermDrawerVisible(false);
      } else {
        message.error((res as any).message || intl.formatMessage({ id: 'role.saveFail' }));
      }
    } catch {
    } finally {
      setSavingPerms(false);
    }
  };

  const { run: handleDelete } = useRequest(
    async (id: string) => {
      const response = await deleteRole(id);
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'common.deleteOk' }));
        tableRef.current?.reload();
        return true;
      } else {
        message.error(response.message || intl.formatMessage({ id: 'common.deleteFail' }));
        return false;
      }
    },
    { manual: true }
  );

  const columns: ProColumns<API.Role>[] = useMemo(
    () => [
      { title: intl.formatMessage({ id: 'common.id' }), dataIndex: 'id', width: 80 },
      { title: intl.formatMessage({ id: 'role.col.name' }), dataIndex: 'name', width: 150 },
      {
        title: intl.formatMessage({ id: 'role.col.code' }),
        dataIndex: 'code',
        width: 150,
        render: (code: any) => (
          <Text code style={{ fontSize: 12 }}>
            {code}
          </Text>
        ),
      },
      { title: intl.formatMessage({ id: 'common.description' }), dataIndex: 'description', ellipsis: true },
      {
        title: intl.formatMessage({ id: 'common.createdAt' }),
        dataIndex: 'createdAt',
        width: 180,
        valueType: 'dateTime',
        hideInSearch: true,
      },
      {
        title: intl.formatMessage({ id: 'home.col.action' }),
        valueType: 'option',
        width: 88,
        fixed: 'right',
        align: 'center',
        render: (_, record) => {
          const items: MenuProps['items'] = [
            {
              key: 'perm',
              label: intl.formatMessage({ id: 'role.assignPerm' }),
              icon: <SafetyOutlined />,
              onClick: () => openPermDrawer(record),
            },
            {
              key: 'instance',
              label: intl.formatMessage({ id: 'role.instancePerm' }),
              icon: <CloudServerOutlined />,
              onClick: () => {
                setInstRole(record);
                setInstDrawerVisible(true);
              },
            },
            {
              key: 'edit',
              label: intl.formatMessage({ id: 'common.edit' }),
              icon: <EditOutlined />,
              onClick: () => {
                setEditingRecord(record);
                setEditModalVisible(true);
              },
            },
            { type: 'divider' },
            {
              key: 'del',
              danger: true,
              label: intl.formatMessage({ id: 'common.delete' }),
              icon: <DeleteOutlined />,
              onClick: () => {
                Modal.confirm({
                  title: intl.formatMessage({ id: 'role.confirmDelete' }),
                  okText: intl.formatMessage({ id: 'common.delete' }),
                  okButtonProps: { danger: true },
                  onOk: () => handleDelete(record.id),
                });
              },
            },
          ];
          return (
            <Dropdown menu={{ items }} trigger={['click']}>
              <Button type="link" size="small" icon={<EllipsisOutlined />}>
                {intl.formatMessage({ id: 'home.col.action' })}
              </Button>
            </Dropdown>
          );
        },
      },
    ],
    [intl]
  );

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'pageTitle.permissionRoles' })}
      extra={[
        <Button
          key="create"
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setCreateModalVisible(true)}
        >
          {intl.formatMessage({ id: 'role.addRole' })}
        </Button>,
      ]}
    >
      <ProTable<API.Role>
        actionRef={tableRef}
        columns={columns}
        request={async (params) => {
          const response = await listRoles({ page: params.current || 1, pageSize: params.pageSize || 10 });
          return {
            data: response.list || [],
            success: response.code === 200,
            total: response.pageResponse?.total || 0,
          };
        }}
        rowKey="id"
        search={{ labelWidth: 'auto' }}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
        scroll={{ x: 900 }}
      />

      {/* Permission assignment drawer */}
      <Drawer
        title={
          permRole
            ? intl.formatMessage({ id: 'role.permDrawerTitle' }, { name: permRole.name })
            : intl.formatMessage({ id: 'role.permDrawerPlain' })
        }
        width={520}
        open={permDrawerVisible}
        onClose={() => setPermDrawerVisible(false)}
        extra={
          <Space>
            <Button onClick={() => setPermDrawerVisible(false)}>
              {intl.formatMessage({ id: 'common.cancel' })}
            </Button>
            <Button type="primary" loading={savingPerms} onClick={handleSavePerms}>
              {intl.formatMessage({ id: 'common.save' })}
            </Button>
          </Space>
        }
      >
        {loadingPerms ? (
          <div style={{ textAlign: 'center', paddingTop: 60 }}>
            <Spin tip={intl.formatMessage({ id: 'role.spinLoading' })} />
          </div>
        ) : (
          <Tree
            checkable
            selectable={false}
            treeData={treeData}
            checkedKeys={checkedKeys}
            onCheck={(checked) => setCheckedKeys(checked as string[])}
            defaultExpandAll
            style={{ fontSize: 13 }}
          />
        )}
      </Drawer>

      <ModalForm
        title={intl.formatMessage({ id: 'role.addRole' })}
        open={createModalVisible}
        onOpenChange={setCreateModalVisible}
        onFinish={async (values) => {
          const response = await createRole(values as API.CreateRoleRequest);
          if (response.code === 200) {
            message.success(intl.formatMessage({ id: 'user.addOk' }));
            setCreateModalVisible(false);
            tableRef.current?.reload();
            return true;
          }
          message.error(response.message || intl.formatMessage({ id: 'user.addFail' }));
          return false;
        }}
        width={600}
      >
        <ProFormText
          name="name"
          label={intl.formatMessage({ id: 'role.col.name' })}
          rules={[{ required: true, message: intl.formatMessage({ id: 'role.enterName' }) }]}
        />
        <ProFormText
          name="code"
          label={intl.formatMessage({ id: 'role.col.code' })}
          rules={[{ required: true, message: intl.formatMessage({ id: 'role.enterCode' }) }]}
          extra={intl.formatMessage({ id: 'role.codeExtra' })}
        />
        <ProFormTextArea
          name="description"
          label={intl.formatMessage({ id: 'common.description' })}
          fieldProps={{ rows: 3 }}
        />
      </ModalForm>

      <ModalForm
        title={intl.formatMessage({ id: 'role.editRole' })}
        open={editModalVisible}
        onOpenChange={setEditModalVisible}
        initialValues={editingRecord || {}}
        onFinish={async (values) => {
          if (!editingRecord) return false;
          const response = await updateRole(editingRecord.id, values);
          if (response.code === 200) {
            message.success(intl.formatMessage({ id: 'common.updateOk' }));
            setEditModalVisible(false);
            setEditingRecord(null);
            tableRef.current?.reload();
            return true;
          }
          message.error(response.message || intl.formatMessage({ id: 'common.operationFailed' }));
          return false;
        }}
        width={600}
      >
        <ProFormText name="name" label={intl.formatMessage({ id: 'role.col.name' })} />
        <ProFormTextArea
          name="description"
          label={intl.formatMessage({ id: 'common.description' })}
          fieldProps={{ rows: 3 }}
        />
      </ModalForm>

      <InstancePermissionsDrawer
        open={instDrawerVisible}
        onClose={() => {
          setInstDrawerVisible(false);
          setInstRole(null);
        }}
        mode="role"
        targetId={instRole?.id ? String(instRole.id) : ''}
        targetLabel={instRole?.name}
      />
    </PageContainer>
  );
};
