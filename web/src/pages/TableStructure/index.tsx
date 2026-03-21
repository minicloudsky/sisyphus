import { PageContainer } from '@ant-design/pro-components';
import {
  Card, Select, Space, Typography, Table, Tag, Tabs, Button,
  List, Alert, message, Divider, Row, Col
} from 'antd';
import {
  DatabaseOutlined, TableOutlined, BulbOutlined,
  SearchOutlined, ReloadOutlined
} from '@ant-design/icons';
import {
  listInstances, listDatabases, listTables, getTableDDL,
  getTableColumns, getAdvisorSuggestions, getExplain
} from '@/services/api';
import { useRequest } from 'ahooks';
import { useState, useMemo } from 'react';
import Editor from '@monaco-editor/react';
import { useIntl } from '@/locales';

const { Text, Paragraph } = Typography;

const levelColor: Record<string, string> = {
  error: 'error',
  warning: 'warning',
  info: 'processing',
};

export default () => {
  const intl = useIntl();
  const [instanceId, setInstanceId] = useState<string>('');
  const [database, setDatabase] = useState<string>('');
  const [table, setTable] = useState<string>('');
  const [ddl, setDDL] = useState<string>('');
  const [columns, setColumns] = useState<any[]>([]);
  const [suggestions, setSuggestions] = useState<any[]>([]);
  const [sql, setSql] = useState<string>('');
  const [explainResult, setExplainResult] = useState<string>('');
  const [advisorLoading, setAdvisorLoading] = useState(false);
  const [explainLoading, setExplainLoading] = useState(false);

  const { data: instancesData } = useRequest(() =>
    listInstances({ page: 1, pageSize: 1000, instancePermissionScope: 'query' })
  );
  const instances = instancesData?.list || [];

  const { data: dbData, run: fetchDatabases, loading: dbLoading } = useRequest(
    () => listDatabases(instanceId),
    {
      manual: true,
      onError: () => {},
    }
  );
  const databases = dbData?.databases || [];

  const { data: tableData, run: fetchTables, loading: tableLoading } = useRequest(
    () => listTables(instanceId, database),
    {
      manual: true,
      onError: () => {},
    }
  );
  const tables = tableData?.tables || [];

  const handleInstanceChange = (val: string) => {
    setInstanceId(val);
    setDatabase('');
    setTable('');
    setDDL('');
    setColumns([]);
    setSuggestions([]);
    fetchDatabases();
  };

  const handleDatabaseChange = (val: string) => {
    setDatabase(val);
    setTable('');
    setDDL('');
    setColumns([]);
    setSuggestions([]);
    fetchTables();
  };

  const handleTableChange = async (val: string) => {
    setTable(val);
    setSuggestions([]);
    setExplainResult('');

    try {
      const [ddlRes, colRes] = await Promise.all([
        getTableDDL(instanceId, database, val),
        getTableColumns(instanceId, database, val),
      ]);
      setDDL(ddlRes?.ddl || '');
      setColumns(colRes?.columns || []);
    } catch {
    }
  };

  const handleGetAdvisor = async () => {
    if (!sql.trim()) {
      message.warning(intl.formatMessage({ id: 'tableStruct.warnSql' }));
      return;
    }
    setAdvisorLoading(true);
    try {
      const res = await getAdvisorSuggestions(instanceId, { database, table, sql });
      if (res?.code === 200) {
        setSuggestions(res.suggestions || []);
      } else {
        message.error(res?.message || intl.formatMessage({ id: 'queryMain.advisorFail' }));
      }
    } catch {
    } finally {
      setAdvisorLoading(false);
    }
  };

  const handleGetExplain = async () => {
    if (!sql.trim()) {
      message.warning(intl.formatMessage({ id: 'tableStruct.warnSql' }));
      return;
    }
    setExplainLoading(true);
    try {
      const res = await getExplain(instanceId, database, sql);
      if (res?.code === 200) {
        setExplainResult(typeof res.explain === 'string' ? res.explain : JSON.stringify(res.explain, null, 2));
      } else {
        message.error(res?.message || intl.formatMessage({ id: 'queryMain.explainFail' }));
      }
    } catch {
    } finally {
      setExplainLoading(false);
    }
  };

  const columnDefs = useMemo(
    () => [
      { title: intl.formatMessage({ id: 'queryMain.struct.col.name' }), dataIndex: 'name', key: 'name', width: 150 },
      { title: intl.formatMessage({ id: 'queryMain.struct.col.type' }), dataIndex: 'type', key: 'type', width: 150,
        render: (t: string) => <Text code>{t}</Text> },
      { title: intl.formatMessage({ id: 'queryMain.struct.col.nullable' }), dataIndex: 'null', key: 'null', width: 70,
        render: (v: string) => v === 'YES'
          ? <Tag color="default">{intl.formatMessage({ id: 'common.yes' })}</Tag>
          : <Tag color="green">{intl.formatMessage({ id: 'common.no' })}</Tag> },
      { title: intl.formatMessage({ id: 'queryMain.struct.col.key' }), dataIndex: 'key', key: 'key', width: 60,
        render: (v: string) => v ? <Tag color={v === 'PRI' ? 'gold' : v === 'UNI' ? 'blue' : 'default'}>{v}</Tag> : null },
      { title: intl.formatMessage({ id: 'queryMain.struct.col.default' }), dataIndex: 'default', key: 'default', width: 120,
        render: (v: string) => v || <Text type="secondary">NULL</Text> },
      { title: intl.formatMessage({ id: 'queryMain.struct.col.comment' }), dataIndex: 'comment', key: 'comment', ellipsis: true },
    ],
    [intl]
  );

  return (
    <PageContainer title={intl.formatMessage({ id: 'tableStruct.pageTitle' })}>
      <Card>
        <Row gutter={[12, 12]}>
          <Col flex="auto">
            <Space wrap size={[8, 8]} style={{ width: '100%' }}>
              <DatabaseOutlined style={{ color: '#1677ff' }} />
              <Select
                style={{ width: 220, minWidth: 160 }}
                placeholder={intl.formatMessage({ id: 'tableStruct.pickInst' })}
                value={instanceId || undefined}
                onChange={handleInstanceChange}
                showSearch
                options={instances.map((inst: any) => ({
                  label: `${inst.name} (${inst.dbType || inst.db_type})`,
                  value: String(inst.id),
                }))}
              />
              <Select
                style={{ width: 150, minWidth: 120 }}
                placeholder={intl.formatMessage({ id: 'tableStruct.pickDb' })}
                value={database || undefined}
                onChange={handleDatabaseChange}
                loading={dbLoading}
                disabled={!instanceId}
                showSearch
                options={databases.map((db: string) => ({ label: db, value: db }))}
              />
              <Select
                style={{ width: 180, minWidth: 140 }}
                placeholder={intl.formatMessage({ id: 'tableStruct.pickTable' })}
                value={table || undefined}
                onChange={handleTableChange}
                loading={tableLoading}
                disabled={!database}
                showSearch
                options={tables.map((t: string) => ({ label: t, value: t }))}
              />
            </Space>
          </Col>
        </Row>

        {table && (
          <div style={{ marginTop: 16 }}>
            <Tabs
              items={[
                {
                  key: 'columns',
                  label: <><TableOutlined /> {intl.formatMessage({ id: 'queryMain.structColInfo' })}</>,
                  children: (
                    <Table
                      columns={columnDefs}
                      dataSource={columns}
                      rowKey="name"
                      size="small"
                      pagination={false}
                      scroll={{ y: 400 }}
                    />
                  ),
                },
                {
                  key: 'ddl',
                  label: intl.formatMessage({ id: 'queryMain.structDdl' }),
                  children: (
                    <Editor
                      height="350px"
                      defaultLanguage="sql"
                      value={ddl}
                      theme="vs"
                      options={{ readOnly: true, minimap: { enabled: false }, fontSize: 13 }}
                    />
                  ),
                },
                {
                  key: 'advisor',
                  label: <><BulbOutlined /> {intl.formatMessage({ id: 'queryMain.advisorTitle' })}</>,
                  children: (
                    <div>
                      <Paragraph type="secondary" style={{ marginBottom: 8 }}>
                        {intl.formatMessage({ id: 'tableStruct.hintSql' }, { table })}
                      </Paragraph>
                      <Editor
                        height="150px"
                        defaultLanguage="sql"
                        value={sql}
                        onChange={(v) => setSql(v || '')}
                        theme="vs"
                        options={{ minimap: { enabled: false }, fontSize: 13 }}
                      />
                      <Space style={{ marginTop: 8, marginBottom: 16 }}>
                        <Button
                          type="primary"
                          icon={<BulbOutlined />}
                          loading={advisorLoading}
                          onClick={handleGetAdvisor}
                          disabled={!instanceId || !database}
                        >
                          {intl.formatMessage({ id: 'tableStruct.btn.advisor' })}
                        </Button>
                        <Button
                          icon={<SearchOutlined />}
                          loading={explainLoading}
                          onClick={handleGetExplain}
                          disabled={!instanceId || !database}
                        >
                          {intl.formatMessage({ id: 'tableStruct.btn.explain' })}
                        </Button>
                        {suggestions.length > 0 && (
                          <Button icon={<ReloadOutlined />} onClick={() => { setSuggestions([]); setExplainResult(''); }}>
                            {intl.formatMessage({ id: 'tableStruct.btn.clear' })}
                          </Button>
                        )}
                      </Space>

                      {suggestions.length > 0 && (
                        <div>
                          <Divider orientation="left">{intl.formatMessage({ id: 'tableStruct.divider.advisor' })}</Divider>
                          <List
                            size="small"
                            dataSource={suggestions}
                            renderItem={(item: any) => (
                              <List.Item>
                                <Alert
                                  type={item.level === 'error' ? 'error' : item.level === 'warning' ? 'warning' : 'info'}
                                  message={
                                    <Space>
                                      <Tag color={levelColor[item.level] || 'default'}>{item.rule}</Tag>
                                      {item.message}
                                    </Space>
                                  }
                                  description={item.detail}
                                  style={{ width: '100%' }}
                                  showIcon
                                />
                              </List.Item>
                            )}
                          />
                        </div>
                      )}

                      {explainResult && (
                        <div>
                          <Divider orientation="left">{intl.formatMessage({ id: 'tableStruct.divider.explain' })}</Divider>
                          <Editor
                            height="200px"
                            defaultLanguage="json"
                            value={explainResult}
                            theme="vs"
                            options={{ readOnly: true, minimap: { enabled: false }, fontSize: 12 }}
                          />
                        </div>
                      )}
                    </div>
                  ),
                },
              ]}
            />
          </div>
        )}

        {!table && instanceId && (
          <div style={{ marginTop: 24, textAlign: 'center', color: '#999' }}>
            {intl.formatMessage({ id: 'tableStruct.empty.pickTable' })}
          </div>
        )}
        {!instanceId && (
          <div style={{ marginTop: 24, textAlign: 'center', color: '#999' }}>
            {intl.formatMessage({ id: 'tableStruct.empty.pickInst' })}
          </div>
        )}
      </Card>
    </PageContainer>
  );
};
