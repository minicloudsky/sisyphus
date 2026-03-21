import { PageContainer, ProForm, ProFormSelect, ProFormDigit } from '@ant-design/pro-components';
import { Card, Button, Table, message, Space, Typography, Alert } from 'antd';
import { hbaseQuery, listInstances } from '@/services/api';
import { useRequest } from 'ahooks';
import { useState } from 'react';
import { PlayCircleOutlined, QuestionCircleOutlined } from '@ant-design/icons';
import Editor from '@monaco-editor/react';
import { useIntl } from '@/locales';

const { Text, Paragraph } = Typography;

const SQL_EXAMPLES = [
  "SELECT * FROM user_info WHERE rowkey = 'user_001'",
  "SELECT cf1:name, cf1:age FROM user_table WHERE rowkey IN ('u1','u2','u3')",
  "SELECT * FROM order_table WHERE rowkey >= 'ord_20240101' AND rowkey <= 'ord_20240201' LIMIT 100",
  "SELECT * FROM logs WHERE rowkey >= 'log_' AND cf:level = 'ERROR' LIMIT 50",
];

export default () => {
  const intl = useIntl();
  const [sql, setSql] = useState(SQL_EXAMPLES[0]);
  const [queryResult, setQueryResult] = useState<{ columns: string[]; rows: any[][] } | null>(null);
  const [loading, setLoading] = useState(false);

  const { data: instancesData } = useRequest(() =>
    listInstances({ page: 1, pageSize: 1000, dbType: 'hbase', instancePermissionScope: 'query' })
  );
  const instances = instancesData?.list || [];

  const handleQuery = async (values: { instanceId: string; limit: number }) => {
    if (!sql.trim()) {
      message.warning(intl.formatMessage({ id: 'queryEngine.warnEnterQuery' }));
      return;
    }

    setLoading(true);
    try {
      const response = await hbaseQuery({
        instanceId: values.instanceId,
        sql: sql.trim(),
        limit: values.limit || 1000,
      });
      if (response.code === 200 && response.data) {
        const rows = (response.data.rows || []).map((row: any[], idx: number) => {
          const obj: Record<string, string> = { _key: String(idx) };
          (response.data!.columns || []).forEach((col: string, i: number) => {
            obj[col] = String(row[i] ?? '');
          });
          return obj;
        });
        setQueryResult({ columns: response.data.columns || [], rows: rows as any });
        message.success(intl.formatMessage({ id: 'queryEngine.successRows' }, { rows: rows.length }));
      } else {
        message.error(response.message || intl.formatMessage({ id: 'queryEngine.fail' }));
      }
    } catch {
    } finally {
      setLoading(false);
    }
  };

  return (
    <PageContainer title={intl.formatMessage({ id: 'queryEngine.title.hbase' })}>
      <Alert
        message={intl.formatMessage({ id: 'queryEngine.hbase.syntaxTitle' })}
        description={
          <div>
            <Paragraph style={{ marginBottom: 4 }}>
              {intl.formatMessage({ id: 'queryEngine.hbase.modes' })}
            </Paragraph>
            <ul style={{ margin: 0, paddingLeft: 20 }}>
              <li>
                <Text code>WHERE rowkey = &apos;value&apos;</Text> — {intl.formatMessage({ id: 'queryEngine.hbase.li1' })}
              </li>
              <li>
                <Text code>WHERE rowkey IN (&apos;v1&apos;,&apos;v2&apos;)</Text> —{' '}
                {intl.formatMessage({ id: 'queryEngine.hbase.li2' })}
              </li>
              <li>
                <Text code>
                  WHERE rowkey &gt;= &apos;start&apos; AND rowkey &lt;= &apos;end&apos;
                </Text>{' '}
                — {intl.formatMessage({ id: 'queryEngine.hbase.li3' })}
              </li>
              <li>
                <Text code>AND cf:qualifier = &apos;value&apos;</Text> —{' '}
                {intl.formatMessage({ id: 'queryEngine.hbase.li4' })}
              </li>
              <li>
                {intl.formatMessage({ id: 'queryEngine.hbase.li5' })}
                <Text code>family:qualifier</Text>
                {intl.formatMessage({ id: 'queryEngine.hbase.li5b' })}
                <Text code>*</Text>
                {intl.formatMessage({ id: 'queryEngine.hbase.li5c' })}
              </li>
            </ul>
          </div>
        }
        type="info"
        showIcon
        icon={<QuestionCircleOutlined />}
        style={{ marginBottom: 16 }}
        closable
      />

      <Card>
        <ProForm
          onFinish={handleQuery}
          initialValues={{ limit: 1000 }}
          submitter={{
            render: () => (
              <Space>
                <Button type="primary" htmlType="submit" icon={<PlayCircleOutlined />} loading={loading}>
                  {intl.formatMessage({ id: 'queryEngine.run' })}
                </Button>
              </Space>
            ),
          }}
        >
          <ProFormSelect
            name="instanceId"
            label={intl.formatMessage({ id: 'queryEngine.col.hbaseInst' })}
            placeholder={intl.formatMessage({ id: 'queryEngine.ph.hbaseInst' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'queryEngine.rule.hbaseInst' }) }]}
            options={instances.map((inst: any) => ({
              label: `${inst.name} (${inst.host}:${inst.port})`,
              value: inst.id,
            }))}
            fieldProps={{ showSearch: true }}
            extra={intl.formatMessage({ id: 'queryEngine.hbase.extra' })}
          />

          <ProFormDigit
            name="limit"
            label={intl.formatMessage({ id: 'queryEngine.col.maxRows' })}
            min={1}
            max={10000}
            initialValue={1000}
            fieldProps={{ style: { width: 160 } }}
          />

          <ProForm.Item
            name="sql"
            label={intl.formatMessage({ id: 'queryEngine.col.querySql' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'queryEngine.rule.querySql' }) }]}
          >
            <div>
              <Space style={{ marginBottom: 8 }} wrap>
                {SQL_EXAMPLES.map((example, idx) => (
                  <Button key={idx} size="small" onClick={() => setSql(example)}>
                    {intl.formatMessage({ id: 'queryEngine.exampleN' }, { n: idx + 1 })}
                  </Button>
                ))}
              </Space>
              <Editor
                height="180px"
                defaultLanguage="sql"
                value={sql}
                onChange={(value) => setSql(value || '')}
                theme="vs"
                options={{
                  minimap: { enabled: false },
                  fontSize: 14,
                  wordWrap: 'on',
                }}
              />
            </div>
          </ProForm.Item>
        </ProForm>
      </Card>

      {queryResult && (
        <Card
          title={intl.formatMessage({ id: 'queryEngine.resultRows' }, { rows: queryResult.rows.length })}
          style={{ marginTop: 16 }}
        >
          <Table
            columns={queryResult.columns.map((col) => ({
              title: col,
              dataIndex: col,
              key: col,
              ellipsis: true,
              width: col === 'rowkey' ? 200 : undefined,
            }))}
            dataSource={queryResult.rows}
            rowKey="_key"
            scroll={{ x: 'max-content' }}
            size="small"
            pagination={{ pageSize: 50, showSizeChanger: true, showQuickJumper: true }}
          />
        </Card>
      )}
    </PageContainer>
  );
};
