import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Tag, Modal } from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import { listAnalysisTasks, getAnalysisTask } from '@/services/api';
import type { ProColumns } from '@ant-design/pro-components';
import { useRequest } from 'ahooks';
import { useState, useMemo } from 'react';
import { useIntl } from '@/locales';

export default () => {
  const intl = useIntl();
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [currentTask, setCurrentTask] = useState<API.RedisAnalysisTask | null>(null);

  const { run: loadDetail } = useRequest(
    async (id: string) => {
      const response = await getAnalysisTask(id);
      if (response.code === 200 && response.data) {
        setCurrentTask(response.data);
        setDetailModalVisible(true);
      }
    },
    { manual: true }
  );

  const statusMap = useMemo(
    () =>
      ({
        0: { text: intl.formatMessage({ id: 'redis.task.status.running' }), color: 'blue' },
        1: { text: intl.formatMessage({ id: 'redis.task.status.done' }), color: 'green' },
        2: { text: intl.formatMessage({ id: 'redis.task.status.fail' }), color: 'red' },
      }) as Record<number, { text: string; color: string }>,
    [intl]
  );

  const columns: ProColumns<API.RedisAnalysisTask>[] = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'redis.tasks.col.id' }),
        dataIndex: 'id',
        width: 100,
      },
      {
        title: intl.formatMessage({ id: 'redis.tasks.col.instId' }),
        dataIndex: 'instanceId',
        width: 100,
      },
      {
        title: intl.formatMessage({ id: 'redis.tasks.col.type' }),
        dataIndex: 'taskType',
        width: 120,
        valueEnum: {
          hotkey: { text: intl.formatMessage({ id: 'redis.tasks.type.hot' }) },
          bigkey: { text: intl.formatMessage({ id: 'redis.tasks.type.big' }) },
        },
      },
      {
        title: intl.formatMessage({ id: 'instance.col.status' }),
        dataIndex: 'status',
        width: 100,
        valueEnum: statusMap,
        render: (_, record) => {
          const status = statusMap[record.status];
          return <Tag color={status.color}>{status.text}</Tag>;
        },
      },
      {
        title: intl.formatMessage({ id: 'redis.tasks.col.created' }),
        dataIndex: 'createdAt',
        width: 180,
        valueType: 'dateTime',
      },
      {
        title: intl.formatMessage({ id: 'home.col.action' }),
        valueType: 'option',
        width: 100,
        render: (_, record) => (
          <a onClick={() => loadDetail(record.id)}>
            <EyeOutlined /> {intl.formatMessage({ id: 'common.view' })}
          </a>
        ),
      },
    ],
    [intl, statusMap, loadDetail]
  );

  return (
    <PageContainer title={intl.formatMessage({ id: 'redis.tasks.title' })}>
      <ProTable<API.RedisAnalysisTask>
        columns={columns}
        request={async (params) => {
          const response = await listAnalysisTasks({
            page: params.current || 1,
            pageSize: params.pageSize || 10,
            instanceId: params.instanceId,
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
      />

      <Modal
        title={intl.formatMessage({ id: 'redis.tasks.detail' })}
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={null}
        width={800}
      >
        {currentTask && (
          <div>
            <p><strong>{intl.formatMessage({ id: 'redis.tasks.detail.id' })}</strong> {currentTask.id}</p>
            <p>
              <strong>{intl.formatMessage({ id: 'redis.tasks.detail.type' })}</strong>{' '}
              {currentTask.taskType === 'hotkey'
                ? intl.formatMessage({ id: 'redis.tasks.type.hot' })
                : intl.formatMessage({ id: 'redis.tasks.type.big' })}
            </p>
            <p>
              <strong>{intl.formatMessage({ id: 'redis.tasks.detail.status' })}</strong>{' '}
              <Tag color={statusMap[currentTask.status].color}>{statusMap[currentTask.status].text}</Tag>
            </p>
            {currentTask.result && (
              <div>
                <strong>{intl.formatMessage({ id: 'redis.tasks.detail.result' })}</strong>
                <pre style={{ background: '#f5f5f5', padding: '12px', borderRadius: '4px', maxHeight: '400px', overflow: 'auto' }}>
                  {JSON.stringify(JSON.parse(currentTask.result), null, 2)}
                </pre>
              </div>
            )}
            {currentTask.errorMessage && (
              <p style={{ color: '#ff4d4f' }}>
                <strong>{intl.formatMessage({ id: 'redis.tasks.detail.err' })}</strong> {currentTask.errorMessage}
              </p>
            )}
          </div>
        )}
      </Modal>
    </PageContainer>
  );
};
