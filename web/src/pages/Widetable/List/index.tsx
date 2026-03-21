import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Button, Space, Tag, message, Popconfirm, Input } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, EyeOutlined, PlayCircleOutlined, SyncOutlined } from '@ant-design/icons';
import { listWidetableConfigs, deleteWidetableConfig } from '@/services/api';
import type { ProColumns } from '@ant-design/pro-components';
import { useNavigate } from 'react-router-dom';
import { useRef, useState, useMemo } from 'react';
import type { ActionType } from '@ant-design/pro-components';
import { useIntl } from '@/locales';

interface WidetableConfig {
  id: number;
  name: string;
  description: string;
  source_instance_id: number;
  main_database: string;
  main_table: string;
  join_config: string;
  field_config: string;
  sync_condition: string;
  target_instance_id: number;
  target_database: string;
  target_table: string;
  status: number;
  created_at: string;
  updated_at: string;
}

export default () => {
  const intl = useIntl();
  const navigate = useNavigate();
  const actionRef = useRef<ActionType>();
  const [keyword, setKeyword] = useState('');

  const statusMap = useMemo(
    () =>
      ({
        0: { text: intl.formatMessage({ id: 'common.disabled' }), color: 'default' },
        1: { text: intl.formatMessage({ id: 'common.enabled' }), color: 'success' },
      }) as Record<number, { text: string; color: string }>,
    [intl]
  );

  const handleDelete = async (id: number) => {
    try {
      const response = await deleteWidetableConfig(id);
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'common.deleteOk' }));
        actionRef.current?.reload();
      } else {
        message.error(response.message || intl.formatMessage({ id: 'common.deleteFail' }));
      }
    } catch {
    }
  };

  const columns: ProColumns<WidetableConfig>[] = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'common.id' }),
        dataIndex: 'id',
        width: 70,
        hideInSearch: true,
      },
      {
        title: intl.formatMessage({ id: 'common.name' }),
        dataIndex: 'name',
        width: 180,
        hideInSearch: true,
        render: (_, record) => (
          <a onClick={() => navigate(`/widetable/detail/${record.id}`)}>{record.name}</a>
        ),
      },
      {
        title: intl.formatMessage({ id: 'common.description' }),
        dataIndex: 'description',
        width: 220,
        hideInSearch: true,
        ellipsis: true,
      },
      {
        title: intl.formatMessage({ id: 'widetable.col.sourceInst' }),
        dataIndex: 'source_instance_id',
        width: 110,
        hideInSearch: true,
      },
      {
        title: intl.formatMessage({ id: 'widetable.col.mainTable' }),
        dataIndex: 'main_table',
        width: 200,
        hideInSearch: true,
        render: (_, record) =>
          record.main_database
            ? `${record.main_database}.${record.main_table}`
            : record.main_table,
      },
      {
        title: intl.formatMessage({ id: 'instance.col.status' }),
        dataIndex: 'status',
        width: 90,
        hideInSearch: true,
        render: (_, record) => {
          const s = statusMap[record.status] || {
            text: intl.formatMessage({ id: 'widetable.unknownStatus' }, { status: record.status }),
            color: 'default',
          };
          return <Tag color={s.color}>{s.text}</Tag>;
        },
      },
      {
        title: intl.formatMessage({ id: 'common.createdAt' }),
        dataIndex: 'created_at',
        width: 180,
        hideInSearch: true,
      },
      {
        title: intl.formatMessage({ id: 'home.col.action' }),
        valueType: 'option',
        width: 280,
        fixed: 'right',
        render: (_, record) => (
          <Space>
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              onClick={() => navigate(`/widetable/detail/${record.id}`)}
            >
              {intl.formatMessage({ id: 'widetable.detail' })}
            </Button>
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => navigate(`/widetable/create?id=${record.id}`)}
            >
              {intl.formatMessage({ id: 'common.edit' })}
            </Button>
            <Button
              type="link"
              size="small"
              icon={<PlayCircleOutlined />}
              onClick={() => navigate(`/widetable/detail/${record.id}?tab=query`)}
            >
              {intl.formatMessage({ id: 'widetable.query' })}
            </Button>
            <Button
              type="link"
              size="small"
              icon={<SyncOutlined />}
              onClick={() => navigate(`/widetable/detail/${record.id}?tab=sync`)}
            >
              {intl.formatMessage({ id: 'widetable.sync' })}
            </Button>
            <Popconfirm
              title={intl.formatMessage({ id: 'widetable.confirmDelete' })}
              onConfirm={() => handleDelete(record.id)}
            >
              <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                {intl.formatMessage({ id: 'common.delete' })}
              </Button>
            </Popconfirm>
          </Space>
        ),
      },
    ],
    [intl, navigate, statusMap]
  );

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'widetable.title' })}
      extra={[
        <Button
          key="create"
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => navigate('/widetable/create')}
        >
          {intl.formatMessage({ id: 'widetable.newConfig' })}
        </Button>,
      ]}
    >
      <ProTable<WidetableConfig>
        actionRef={actionRef}
        columns={columns}
        rowKey="id"
        search={false}
        toolbar={{
          search: {
            placeholder: intl.formatMessage({ id: 'widetable.searchPh' }),
            onSearch: (value) => {
              setKeyword(value);
              actionRef.current?.reload();
            },
          },
        }}
        request={async (params) => {
          try {
            const response = await listWidetableConfigs({
              page: params.current || 1,
              page_size: params.pageSize || 10,
              keyword: keyword || undefined,
            });
            if (response.code === 200) {
              const data = response.data?.list || response.data || [];
              const total = response.data?.total || response.total || data.length;
              return { data, success: true, total };
            }
            return { data: [], success: false, total: 0 };
          } catch (e) {
            return { data: [], success: false, total: 0 };
          }
        }}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
        scroll={{ x: 1200 }}
      />
    </PageContainer>
  );
};
