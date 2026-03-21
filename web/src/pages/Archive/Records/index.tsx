import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Card, Select, Space, Tag } from 'antd';
import type { ProColumns } from '@ant-design/pro-components';
import { useRequest } from 'ahooks';
import { useMemo, useState } from 'react';
import { listArchiveConfigs, listArchiveRecords } from '@/services/api';
import { useIntl } from '@/locales';
import { formatDateTime } from '@/utils/format';

type ArchiveRecordRow = {
  id?: string;
  config_id?: string;
  work_id?: string;
  status?: number;
  message?: string;
  archived_rows?: number;
  start_time?: string;
  end_time?: string;
  created_at?: string;
};

export default () => {
  const intl = useIntl();
  const [configId, setConfigId] = useState<string>('');

  const { data: cfgRes } = useRequest(() => listArchiveConfigs({ page: 1, pageSize: 500 }));
  const configs = cfgRes?.list || [];

  const columns: ProColumns<ArchiveRecordRow>[] = useMemo(
    () => {
      const statusTag = (s: number) => {
        const map: Record<number, { id: string; color: string }> = {
          0: { id: 'archive.records.status.running', color: 'processing' },
          1: { id: 'archive.records.status.ok', color: 'success' },
          2: { id: 'archive.records.status.fail', color: 'error' },
        };
        const m = map[s] || { id: 'home.status.unknown', color: 'default' };
        return <Tag color={m.color}>{intl.formatMessage({ id: m.id })}</Tag>;
      };
      return [
      { title: intl.formatMessage({ id: 'common.id' }), dataIndex: 'id', width: 90 },
      { title: intl.formatMessage({ id: 'archive.records.col.workId' }), dataIndex: 'work_id', width: 160 },
      {
        title: intl.formatMessage({ id: 'archive.records.col.status' }),
        dataIndex: 'status',
        width: 100,
        render: (v) => statusTag(Number(v)),
      },
      { title: intl.formatMessage({ id: 'archive.records.col.msg' }), dataIndex: 'message', ellipsis: true },
      { title: intl.formatMessage({ id: 'archive.records.col.rows' }), dataIndex: 'archived_rows', width: 110 },
      {
        title: intl.formatMessage({ id: 'archive.records.col.start' }),
        dataIndex: 'start_time',
        width: 180,
        render: (_, r) => formatDateTime((r as ArchiveRecordRow).start_time || '') || '-',
      },
      {
        title: intl.formatMessage({ id: 'archive.records.col.end' }),
        dataIndex: 'end_time',
        width: 180,
        render: (_, r) => formatDateTime((r as ArchiveRecordRow).end_time || '') || '-',
      },
    ];
    },
    [intl]
  );

  return (
    <PageContainer title={intl.formatMessage({ id: 'archive.records.pageTitle' })}>
      <Card style={{ marginBottom: 16 }}>
        <Space>
          <span>{intl.formatMessage({ id: 'archive.col.taskName' })}</span>
          <Select
            style={{ minWidth: 280 }}
            placeholder={intl.formatMessage({ id: 'archive.records.pickConfig' })}
            allowClear
            showSearch
            optionFilterProp="label"
            value={configId || undefined}
            onChange={(v) => setConfigId(v || '')}
            options={configs.map((c: API.ArchiveConfig) => ({
              label: `${c.taskName} (#${c.id})`,
              value: String(c.id),
            }))}
          />
        </Space>
      </Card>
      <ProTable<ArchiveRecordRow>
        key={configId || 'none'}
        rowKey="id"
        search={false}
        columns={columns}
        pagination={{ defaultPageSize: 10, showSizeChanger: true }}
        scroll={{ x: 1100 }}
        request={async (params) => {
          if (!configId) {
            return { data: [], success: true, total: 0 };
          }
          const res = await listArchiveRecords({
            id: configId,
            page: params.current || 1,
            pageSize: params.pageSize || 10,
          } as any);
          const list = (res as any).list || [];
          const total = (res as any).pageResponse?.total ?? list.length;
          return {
            data: list,
            success: (res as any).code === 200,
            total,
          };
        }}
      />
    </PageContainer>
  );
};
