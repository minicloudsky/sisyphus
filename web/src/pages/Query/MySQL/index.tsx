import { PageContainer, ProForm, ProFormSelect, ProFormText } from '@ant-design/pro-components';
import { Card, Button, Table, message, Space, Dropdown, Modal, Input, Typography } from 'antd';
import type { MenuProps } from 'antd';
import { mysqlQuery, exportQuery, downloadQueryExport, listInstances } from '@/services/api';
import { isCommonResponseOk } from '@/utils/kratosResponse';
import { useRequest } from 'ahooks';
import { useState, useMemo, useCallback } from 'react';
import { DownloadOutlined, DownOutlined, PlayCircleOutlined } from '@ant-design/icons';
import { saveAs } from 'file-saver';
import Editor from '@monaco-editor/react';
import { useIntl } from '@/locales';
import './index.less';

export default () => {
  const intl = useIntl();
  const [sql, setSql] = useState('');
  const [queryResult, setQueryResult] = useState<{ columns: string[]; rows: any[][] } | null>(null);
  const [loading, setLoading] = useState(false);
  const [exportInsertModalOpen, setExportInsertModalOpen] = useState(false);
  const [exportInsertTableName, setExportInsertTableName] = useState('query_export');
  const [exportRunning, setExportRunning] = useState(false);

  const { data: instancesData } = useRequest(() =>
    listInstances({ page: 1, pageSize: 1000, dbType: 'mysql', instancePermissionScope: 'query' }),
  );
  const instances = instancesData?.list || [];

  const readFormIds = () => {
    const form = document.querySelector('form');
    const formData = new FormData(form as HTMLFormElement);
    return {
      instanceId: formData.get('instanceId') as string,
      databaseName: formData.get('databaseName') as string,
    };
  };

  const runExport = useCallback(
    async (format: 'excel' | 'csv' | 'json' | 'sql', insertTableName?: string): Promise<boolean> => {
      if (!queryResult) {
        message.warning(intl.formatMessage({ id: 'queryEngine.warnRunFirst' }));
        return false;
      }
      const { instanceId, databaseName } = readFormIds();
      if (!instanceId || !databaseName) {
        message.warning(intl.formatMessage({ id: 'queryEngine.warnPickInstanceDb' }));
        return false;
      }
      try {
        setExportRunning(true);
        const payload: Record<string, unknown> = {
          instanceId,
          database: databaseName,
          sql: sql.trim(),
          format,
        };
        if (format === 'sql') {
          payload.table_name = (insertTableName ?? 'query_export').trim() || 'query_export';
        }
        const res = await exportQuery(payload as any);
        if (!isCommonResponseOk(res)) {
          message.error(res?.message || intl.formatMessage({ id: 'queryEngine.exportFail' }));
          return false;
        }
        const d = res.data;
        const downloadUrl = d?.downloadUrl || d?.download_url;
        if (!downloadUrl) {
          message.error(intl.formatMessage({ id: 'queryEngine.exportFail' }));
          return false;
        }
        const blob = await downloadQueryExport(downloadUrl);
        const ext = format === 'excel' ? 'xlsx' : format;
        const filename =
          d.fileName || d.file_name || `query_result_${Date.now()}.${ext}`;
        saveAs(blob, filename);
        message.success(intl.formatMessage({ id: 'queryEngine.exportOk' }));
        return true;
      } catch {
        return false;
      } finally {
        setExportRunning(false);
      }
    },
    [queryResult, sql, intl],
  );

  const exportMenuItems: MenuProps['items'] = useMemo(
    () => [
      { key: 'excel', label: intl.formatMessage({ id: 'queryMain.export.excel' }) },
      { key: 'csv', label: intl.formatMessage({ id: 'queryMain.export.csv' }) },
      { key: 'json', label: intl.formatMessage({ id: 'queryMain.export.json' }) },
      { type: 'divider' },
      { key: 'sql', label: intl.formatMessage({ id: 'queryMain.export.insertSql' }) },
    ],
    [intl],
  );

  const onExportMenuClick: MenuProps['onClick'] = useCallback(
    ({ key }) => {
      if (key === 'sql') {
        setExportInsertTableName('query_export');
        setExportInsertModalOpen(true);
        return;
      }
      void runExport(key as 'excel' | 'csv' | 'json');
    },
    [runExport],
  );

  const handleQuery = async (values: { instanceId: string; databaseName: string }) => {
    if (!sql.trim()) {
      message.warning(intl.formatMessage({ id: 'queryEngine.warnEnterSql' }));
      return;
    }

    setLoading(true);
    try {
      const response = await mysqlQuery({
        instanceId: (values as any).instanceId,
        database: (values as any).databaseName,
        sql: sql.trim(),
      });
      if (response.code === 200 && response.data) {
        setQueryResult({
          columns: response.data.columns || [],
          rows: response.data.rows || [],
        });
        message.success(
          intl.formatMessage(
            { id: 'queryEngine.successRows' },
            { rows: response.data.rows?.length || 0 }
          )
        );
      } else {
        message.error(response.message || intl.formatMessage({ id: 'queryEngine.fail' }));
      }
    } catch {
    } finally {
      setLoading(false);
    }
  };

  return (
    <PageContainer title={intl.formatMessage({ id: 'queryEngine.title.mysql' })}>
      <Card>
        <ProForm
          onFinish={handleQuery}
          submitter={{
            render: () => (
              <Space>
                <Button type="primary" htmlType="submit" icon={<PlayCircleOutlined />} loading={loading}>
                  {intl.formatMessage({ id: 'queryEngine.run' })}
                </Button>
                {queryResult && (
                  <Dropdown menu={{ items: exportMenuItems, onClick: onExportMenuClick }}>
                    <Button icon={<DownloadOutlined />}>
                      {intl.formatMessage({ id: 'queryEngine.exportMenu' })} <DownOutlined />
                    </Button>
                  </Dropdown>
                )}
              </Space>
            ),
          }}
        >
          <ProFormSelect
            name="instanceId"
            label={intl.formatMessage({ id: 'queryEngine.col.instance' })}
            placeholder={intl.formatMessage({ id: 'queryEngine.ph.instance' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'queryEngine.rule.instance' }) }]}
            options={instances.map((inst) => ({
              label: `${inst.name} (${inst.host}:${inst.port})`,
              value: inst.id,
            }))}
            fieldProps={{
              showSearch: true,
            }}
          />
          <ProFormText
            name="databaseName"
            label={intl.formatMessage({ id: 'queryEngine.col.dbName' })}
            placeholder={intl.formatMessage({ id: 'queryEngine.ph.dbName' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'queryEngine.rule.dbName' }) }]}
          />
          <ProForm.Item
            name="sql"
            label={intl.formatMessage({ id: 'queryEngine.col.sql' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'queryEngine.rule.sql' }) }]}
          >
            <Editor
              height="200px"
              defaultLanguage="sql"
              value={sql}
              onChange={(value) => setSql(value || '')}
              theme="vs"
              options={{
                minimap: { enabled: false },
                fontSize: 14,
              }}
            />
          </ProForm.Item>
        </ProForm>
      </Card>

      <Modal
        title={intl.formatMessage({ id: 'queryMain.export.insertTitle' })}
        open={exportInsertModalOpen}
        onCancel={() => setExportInsertModalOpen(false)}
        confirmLoading={exportRunning}
        destroyOnClose
        onOk={async () => {
          const ok = await runExport('sql', exportInsertTableName);
          if (ok) setExportInsertModalOpen(false);
        }}
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          {intl.formatMessage({ id: 'queryMain.export.tableNameTip' })}
        </Typography.Paragraph>
        <Input
          placeholder="query_export"
          value={exportInsertTableName}
          onChange={(e) => setExportInsertTableName(e.target.value)}
          onPressEnter={async () => {
            const ok = await runExport('sql', exportInsertTableName);
            if (ok) setExportInsertModalOpen(false);
          }}
        />
      </Modal>

      {queryResult && (
        <Card title={intl.formatMessage({ id: 'queryEngine.result' })} style={{ marginTop: 16 }}>
          <Table
            columns={queryResult.columns.map((col) => ({
              title: col,
              dataIndex: col,
              key: col,
              ellipsis: true,
            }))}
            dataSource={queryResult.rows.map((row, index) => {
              const record: any = { key: index };
              queryResult.columns.forEach((col, i) => {
                record[col] = row[i];
              });
              return record;
            })}
            scroll={{ x: 'max-content' }}
            pagination={{
              pageSize: 100,
              showSizeChanger: true,
              showTotal: (total) => intl.formatMessage({ id: 'queryEngine.total' }, { total }),
            }}
          />
        </Card>
      )}
    </PageContainer>
  );
};
