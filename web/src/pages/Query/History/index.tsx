import { PageContainer, ProTable } from '@ant-design/pro-components';
import { Button, Space, message, Popconfirm } from 'antd';
import { DeleteOutlined, EyeOutlined } from '@ant-design/icons';
import { listQueryHistory, deleteQueryHistory } from '@/services/api';
import type { ProColumns } from '@ant-design/pro-components';
import { useRequest } from 'ahooks';
import { useState, useMemo } from 'react';
import Editor from '@monaco-editor/react';
import { Modal } from 'antd';
import { useIntl } from '@/locales';

export default () => {
  const intl = useIntl();
  const [detailModalVisible, setDetailModalVisible] = useState(false);
  const [selectedHistory, setSelectedHistory] = useState<API.QueryHistory | null>(null);

  const { run: handleDelete } = useRequest(
    async (id: string) => {
      const response = await deleteQueryHistory(id);
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

  const columns: ProColumns<API.QueryHistory>[] = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'common.id' }),
        dataIndex: 'id',
        width: 100,
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
        title: intl.formatMessage({ id: 'queryHist.col.execTime' }),
        dataIndex: 'executeTime',
        width: 100,
        render: (time) => `${time}s`,
      },
      {
        title: intl.formatMessage({ id: 'queryHist.col.affected' }),
        dataIndex: 'affectedRows',
        width: 100,
      },
      {
        title: intl.formatMessage({ id: 'queryHist.col.resultCount' }),
        dataIndex: 'resultCount',
        width: 100,
      },
      {
        title: intl.formatMessage({ id: 'queryHist.col.executedAt' }),
        dataIndex: 'createdAt',
        width: 180,
        valueType: 'dateTime',
      },
      {
        title: intl.formatMessage({ id: 'home.col.action' }),
        valueType: 'option',
        width: 150,
        render: (_, record) => (
          <Space>
            <Button
              type="link"
              icon={<EyeOutlined />}
              onClick={() => {
                setSelectedHistory(record);
                setDetailModalVisible(true);
              }}
            >
              {intl.formatMessage({ id: 'common.view' })}
            </Button>
            <Popconfirm
              title={intl.formatMessage({ id: 'queryHist.confirmDel' })}
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
    <PageContainer title={intl.formatMessage({ id: 'queryHist.pageTitle' })}>
      <ProTable<API.QueryHistory>
        columns={columns}
        request={async (params) => {
          const response = await listQueryHistory({
            page: params.current || 1,
            pageSize: params.pageSize || 10,
            instanceId: params.instanceId,
          });
          return {
            data: response.data?.list || [],
            success: response.code === 200,
            total: response.data?.page?.total || 0,
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

      <Modal
        title={intl.formatMessage({ id: 'queryHist.detailTitle' })}
        open={detailModalVisible}
        onCancel={() => setDetailModalVisible(false)}
        footer={null}
        width={800}
      >
        {selectedHistory && (
          <div>
            <p><strong>{intl.formatMessage({ id: 'queryHist.detail.execTime' })}</strong> {selectedHistory.executeTime}s</p>
            <p><strong>{intl.formatMessage({ id: 'queryHist.detail.affected' })}</strong> {selectedHistory.affectedRows}</p>
            <p><strong>{intl.formatMessage({ id: 'queryHist.detail.resultCount' })}</strong> {selectedHistory.resultCount}</p>
            <p><strong>{intl.formatMessage({ id: 'queryHist.detail.time' })}</strong> {selectedHistory.createdAt}</p>
            <div style={{ marginTop: 16 }}>
              <strong>{intl.formatMessage({ id: 'queryHist.detail.sql' })}</strong>
              <Editor
                height="200px"
                defaultLanguage="sql"
                value={selectedHistory.sqlText}
                theme="vs"
                options={{
                  readOnly: true,
                  minimap: { enabled: false },
                }}
              />
            </div>
          </div>
        )}
      </Modal>
    </PageContainer>
  );
};
