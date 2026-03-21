import { PageContainer, ProTable, ModalForm, ProFormText, ProFormTextArea } from '@ant-design/pro-components';
import { Button, Space, Tag, message, Popconfirm } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, PushpinOutlined } from '@ant-design/icons';
import { listFavorites, createFavorite, updateFavorite, deleteFavorite, pinFavorite } from '@/services/api';
import type { ProColumns } from '@ant-design/pro-components';
import { useRequest } from 'ahooks';
import { useState, useMemo } from 'react';
import { useIntl } from '@/locales';
import { isCommonResponseOk } from '@/utils/kratosResponse';

export default () => {
  const intl = useIntl();
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<API.Favorite | null>(null);

  const { run: handleDelete } = useRequest(
    async (id: string) => {
      const response = await deleteFavorite(id);
      if (isCommonResponseOk(response)) {
        message.success(intl.formatMessage({ id: 'common.deleteOk' }));
        return true;
      } else {
        message.error(response.message || intl.formatMessage({ id: 'common.deleteFail' }));
        return false;
      }
    },
    { manual: true }
  );

  const { run: handlePin } = useRequest(
    async (id: string) => {
      const response = await pinFavorite(id);
      if (isCommonResponseOk(response)) {
        message.success(intl.formatMessage({ id: 'queryFav.pinOk' }));
        return true;
      } else {
        message.error(response.message || intl.formatMessage({ id: 'queryFav.pinFail' }));
        return false;
      }
    },
    { manual: true }
  );

  const columns: ProColumns<API.Favorite>[] = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'queryFav.col.title' }),
        dataIndex: 'name',
        width: 200,
        render: (_: unknown, record: API.Favorite & { title?: string }) =>
          record.name || record.title || '—',
      },
      {
        title: intl.formatMessage({ id: 'queryHist.col.instId' }),
        dataIndex: 'instanceId',
        width: 100,
      },
      {
        title: intl.formatMessage({ id: 'home.col.database' }),
        dataIndex: 'databaseName',
        width: 120,
      },
      {
        title: intl.formatMessage({ id: 'queryHist.col.sql' }),
        dataIndex: 'sqlText',
        ellipsis: true,
        width: 300,
      },
      {
        title: intl.formatMessage({ id: 'queryFav.col.desc' }),
        dataIndex: 'description',
        ellipsis: true,
        width: 200,
      },
      {
        title: intl.formatMessage({ id: 'queryFav.pin' }),
        dataIndex: 'isPinned',
        width: 80,
        render: (_, record) => (
          <Tag color={record.isPinned ? 'red' : 'default'}>
            {record.isPinned ? intl.formatMessage({ id: 'common.yes' }) : intl.formatMessage({ id: 'common.no' })}
          </Tag>
        ),
      },
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
        width: 200,
        render: (_, record) => (
          <Space>
            <Button
              type="link"
              icon={<PushpinOutlined />}
              onClick={() => handlePin(record.id)}
            >
              {record.isPinned ? intl.formatMessage({ id: 'queryFav.unpin' }) : intl.formatMessage({ id: 'queryFav.pin' })}
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
              title={intl.formatMessage({ id: 'queryFav.confirmDel' })}
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
      title={intl.formatMessage({ id: 'queryFav.pageTitle' })}
      extra={[
        <Button
          key="create"
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setCreateModalVisible(true)}
        >
          {intl.formatMessage({ id: 'queryFav.add' })}
        </Button>,
      ]}
    >
      <ProTable<API.Favorite>
        columns={columns}
        request={async (params) => {
          const response = await listFavorites({
            page: params.current || 1,
            pageSize: params.pageSize || 10,
          });
          return {
            data: response.favorites || [],
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
        scroll={{ x: 1200 }}
      />

      <ModalForm
        title={intl.formatMessage({ id: 'queryFav.modal.create' })}
        open={createModalVisible}
        onOpenChange={setCreateModalVisible}
        onFinish={async (values) => {
          const v = values as API.CreateFavoriteRequest & { description?: string };
          const { description, ...rest } = v;
          const response = await createFavorite({
            ...rest,
            tag: rest.tag || description,
          });
          if (Number((response as { id?: number }).id) > 0) {
            message.success(intl.formatMessage({ id: 'queryFav.addOk' }));
            setCreateModalVisible(false);
            return true;
          } else {
            message.error(response.message || intl.formatMessage({ id: 'queryFav.addFail' }));
            return false;
          }
        }}
        width={800}
      >
        <ProFormText
          name="name"
          label={intl.formatMessage({ id: 'queryFav.col.title' })}
          rules={[{ required: true, message: intl.formatMessage({ id: 'queryFav.rule.title' }) }]}
        />
        <ProFormText
          name="instanceId"
          label={intl.formatMessage({ id: 'queryHist.col.instId' })}
          rules={[{ required: true, message: intl.formatMessage({ id: 'queryFav.rule.inst' }) }]}
        />
        <ProFormText
          name="databaseName"
          label={intl.formatMessage({ id: 'sqlOrder.create.dbName' })}
          rules={[{ required: true, message: intl.formatMessage({ id: 'queryFav.rule.db' }) }]}
        />
        <ProFormTextArea
          name="sqlText"
          label={intl.formatMessage({ id: 'queryHist.col.sql' })}
          rules={[{ required: true, message: intl.formatMessage({ id: 'queryFav.rule.sql' }) }]}
          fieldProps={{ rows: 6 }}
        />
        <ProFormTextArea
          name="description"
          label={intl.formatMessage({ id: 'queryFav.col.desc' })}
          fieldProps={{ rows: 3 }}
        />
      </ModalForm>

      <ModalForm
        title={intl.formatMessage({ id: 'queryFav.modal.edit' })}
        open={editModalVisible}
        onOpenChange={setEditModalVisible}
        initialValues={
          editingRecord
            ? { ...editingRecord, description: editingRecord.tag ?? '' }
            : {}
        }
        onFinish={async (values) => {
          if (!editingRecord) return false;
          const v = values as { name?: string; sqlText?: string; description?: string };
          const response = await updateFavorite(editingRecord.id, {
            name: v.name ?? '',
            sqlText: v.sqlText ?? '',
            tag: v.description,
          });
          if (isCommonResponseOk(response)) {
            message.success(intl.formatMessage({ id: 'common.updateOk' }));
            setEditModalVisible(false);
            setEditingRecord(null);
            return true;
          } else {
            message.error(response.message || intl.formatMessage({ id: 'common.operationFailed' }));
            return false;
          }
        }}
        width={800}
      >
        <ProFormText
          name="name"
          label={intl.formatMessage({ id: 'queryFav.col.title' })}
          rules={[{ required: true, message: intl.formatMessage({ id: 'queryFav.rule.title' }) }]}
        />
        <ProFormTextArea
          name="sqlText"
          label={intl.formatMessage({ id: 'queryHist.col.sql' })}
          fieldProps={{ rows: 6 }}
        />
        <ProFormTextArea
          name="description"
          label={intl.formatMessage({ id: 'queryFav.col.desc' })}
          fieldProps={{ rows: 3 }}
        />
      </ModalForm>
    </PageContainer>
  );
};
