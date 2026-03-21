import React, { useState, useMemo, useCallback } from 'react';
import { PageContainer } from '@ant-design/pro-components';
import { Card, Form, Select, Button, message, Space, Tag, Table, Modal, Input, Tooltip } from 'antd';
import { useNavigate } from 'react-router-dom';
import Editor from '@monaco-editor/react';
import {
  ReloadOutlined,
  FileTextOutlined,
  FormatPainterOutlined,
  SafetyOutlined,
  SendOutlined,
  DatabaseOutlined,
  TableOutlined,
} from '@ant-design/icons';
import DbTypeIcon from '@/components/DbTypeIcon';
import {
  listInstances,
  listDatabases,
  listTables,
  getTableDDL,
  createSQLOrder,
  reviewSQL,
} from '@/services/api';
import { useRequest } from 'ahooks';
import { useSplitPanePercent } from '@/hooks/useSplitPanePercent';
import './index.less';
import { useIntl } from '@/locales';

const { Option } = Select;
const { TextArea } = Input;

export default () => {
  const intl = useIntl();
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [sqlContent, setSqlContent] = useState('');
  const [tableDDL, setTableDDL] = useState('');
  const [selectedInstanceId, setSelectedInstanceId] = useState<string>('');
  const [selectedDatabase, setSelectedDatabase] = useState<string>('');
  const [selectedTable, setSelectedTable] = useState<string>('');
  const [submitModalVisible, setSubmitModalVisible] = useState(false);
  const [submitForm] = Form.useForm();
  const [auditResult, setAuditResult] = useState<API.SQLReviewRecord[]>([]);
  const [sqlEditorWidth, setSqlEditorWidth] = useState<number>(50); // 百分比，默认50%
  const [isDragging, setIsDragging] = useState(false);
  const {
    percent: leftPanelWidthPct,
    dragging: isLeftPaneDragging,
    onSplitterMouseDown: onLeftSplitterMouseDown,
  } = useSplitPanePercent({
    defaultPct: 11,
    minPct: 9,
    maxPct: 30,
    storageKey: 'sql_order_ddl_sidebar_pct_v2',
  });

  // 获取实例列表
  const { data: instancesData, loading: instancesLoading } = useRequest(
    () => listInstances({ page: 1, pageSize: 1000, dbType: 'mysql', instancePermissionScope: 'ddl' }),
    { refreshDeps: [] }
  );
  const instances = instancesData?.list || [];

  // 获取数据库列表
  const {
    data: databasesData,
    loading: databasesLoading,
    run: loadDatabases,
  } = useRequest(
    (instanceId: string) => listDatabases(instanceId),
    {
      manual: true,
      onError: () => {},
    }
  );
  const databases = databasesData?.databases || [];

  // 获取表列表
  const {
    data: tablesData,
    loading: tablesLoading,
    run: loadTables,
  } = useRequest(
    ({ instanceId, database }: { instanceId: string; database: string }) =>
      listTables(instanceId, database),
    {
      manual: true,
      onError: () => {},
    }
  );
  const tables = tablesData?.tables || [];

  // 获取表DDL
  const {
    loading: ddlLoading,
    run: loadTableDDL,
  } = useRequest(
    ({ instanceId, database, table }: { instanceId: string; database: string; table: string }) =>
      getTableDDL(instanceId, database, table),
    {
      manual: true,
      onSuccess: (data) => {
        if (data.code === 200) {
          setTableDDL(data.ddl || '');
        } else {
          message.error(data.message || intl.formatMessage({ id: 'queryMain.ddlFail' }));
        }
      },
      onError: () => {},
    }
  );

  // 实例选择变化
  const handleInstanceChange = (instanceId: string) => {
    setSelectedInstanceId(instanceId);
    setSelectedDatabase('');
    setSelectedTable('');
    setTableDDL('');
    form.setFieldsValue({ databaseName: '', tableName: '' });
    if (instanceId) {
      loadDatabases(instanceId);
    }
  };

  // 数据库选择变化
  const handleDatabaseChange = (database: string) => {
    setSelectedDatabase(database);
    setSelectedTable('');
    setTableDDL('');
    form.setFieldsValue({ tableName: '' });
    if (selectedInstanceId && database) {
      loadTables({ instanceId: selectedInstanceId, database });
    }
  };

  // 表选择变化
  const handleTableChange = (table: string) => {
    setSelectedTable(table);
    if (selectedInstanceId && selectedDatabase && table) {
      loadTableDDL({ instanceId: selectedInstanceId, database: selectedDatabase, table });
      // 当选择表时，重置SQL编辑器宽度为50%
      setSqlEditorWidth(50);
    } else {
      // 当取消选择表时，SQL编辑器占满
      setSqlEditorWidth(100);
    }
  };

  // SQL美化
  const handleBeautify = () => {
    if (!sqlContent.trim()) {
      message.warning(intl.formatMessage({ id: 'sqlOrder.editor.warnSqlFirst' }));
      return;
    }
    // 简单的SQL美化（可以后续集成更强大的美化工具）
    const beautified = sqlContent
      .replace(/\s+/g, ' ')
      .replace(/\s*,\s*/g, ', ')
      .replace(/\s*\(\s*/g, ' (')
      .replace(/\s*\)\s*/g, ') ')
      .trim();
    setSqlContent(beautified);
    message.success(intl.formatMessage({ id: 'sqlOrder.editor.beautifyOk' }));
  };

  // DDL检测
  const { loading: auditLoading, run: handleDDLCheck } = useRequest(
    async () => {
      if (!sqlContent.trim()) {
        message.warning(intl.formatMessage({ id: 'sqlOrder.editor.enterSql' }));
        return;
      }
      const values = form.getFieldsValue();
      if (!values.instanceId || !values.databaseName) {
        message.warning(intl.formatMessage({ id: 'sqlOrder.editor.warnInstDb' }));
        return;
      }

      // 处理SQL：去除开头和结尾的空格、换行符等空白字符
      const processedSQL = sqlContent.trim().replace(/^[\s\n\r]+|[\s\n\r]+$/g, '');

      if (!processedSQL) {
        message.warning(intl.formatMessage({ id: 'sqlOrder.editor.sqlEmpty' }));
        return;
      }

      const response = await reviewSQL({
        instanceId: values.instanceId,
        databaseName: values.databaseName,
        sqlContent: processedSQL,
      });

      if (response.code === 200) {
        setAuditResult(response.data || []);
        message.success(intl.formatMessage({ id: 'sqlOrder.editor.checkOk' }));
      } else {
        message.error(response.message || intl.formatMessage({ id: 'sqlOrder.editor.checkFail' }));
        setAuditResult([]);
      }
    },
    { manual: true }
  );

  const getStatusText = useCallback(
    (status: string) => {
      const statusMap: Record<string, { id: string; color: string }> = {
        审核完成: { id: 'sqlOrder.stage.auditDone', color: 'success' },
        已审核: { id: 'sqlOrder.stage.checked', color: 'success' },
        执行成功: { id: 'sqlOrder.stage.execOk', color: 'success' },
        已执行: { id: 'sqlOrder.stage.executed', color: 'success' },
        执行中: { id: 'sqlOrder.stage.executing', color: 'processing' },
        审核中: { id: 'sqlOrder.stage.auditing', color: 'processing' },
        备份中: { id: 'sqlOrder.stage.backingUp', color: 'processing' },
        已备份: { id: 'sqlOrder.stage.backedUp', color: 'success' },
        回滚中: { id: 'sqlOrder.stage.rollingBack', color: 'warning' },
        已回滚: { id: 'sqlOrder.stage.rolledBack', color: 'default' },
        执行失败: { id: 'sqlOrder.stage.execFail', color: 'error' },
        执行错误: { id: 'sqlOrder.stage.execErr', color: 'error' },
        待处理: { id: 'sqlOrder.stage.pending', color: 'default' },
        等待中: { id: 'sqlOrder.stage.waiting', color: 'default' },
        'Audit completed': { id: 'sqlOrder.stage.auditDone', color: 'success' },
        CHECKED: { id: 'sqlOrder.stage.checked', color: 'success' },
        EXECUTED: { id: 'sqlOrder.stage.executed', color: 'success' },
        'Execute Successfully': { id: 'sqlOrder.stage.execOk', color: 'success' },
      };
      const mapped = statusMap[status];
      if (mapped) {
        return { text: intl.formatMessage({ id: mapped.id }), color: mapped.color };
      }
      return { text: status || intl.formatMessage({ id: 'sqlOrder.stage.unknown' }), color: 'default' };
    },
    [intl]
  );

  const auditColumns = useMemo(
    () => [
      {
        title: intl.formatMessage({ id: 'sqlOrder.col.seq' }),
        dataIndex: 'sequence',
        key: 'sequence',
        width: 60,
      },
      {
        title: intl.formatMessage({ id: 'sqlOrder.col.sql' }),
        dataIndex: 'sqlText',
        key: 'sqlText',
        width: 350,
        ellipsis: {
          showTitle: false,
        },
        render: (text: string) => {
          if (!text || text.trim() === '') {
            return <span style={{ color: '#999' }}>{intl.formatMessage({ id: 'sqlOrder.sqlStmtParens' })}</span>;
          }
          return (
            <Tooltip title={text} placement="topLeft">
              <span style={{ fontFamily: 'monospace', fontSize: '12px' }}>{text}</span>
            </Tooltip>
          );
        },
      },
      {
        title: intl.formatMessage({ id: 'sqlOrder.col.errLevel' }),
        dataIndex: 'errorLevel',
        key: 'errorLevel',
        width: 100,
        render: (level: number) => {
          if (level === 0) {
            return <Tag color="success">{intl.formatMessage({ id: 'sqlOrder.col.normal' })}</Tag>;
          } else if (level === 1) {
            return <Tag color="warning">{intl.formatMessage({ id: 'sqlOrder.level.warn' })}</Tag>;
          } else {
            return <Tag color="error">{intl.formatMessage({ id: 'sqlOrder.level.err' })}</Tag>;
          }
        },
      },
      {
        title: intl.formatMessage({ id: 'home.col.status' }),
        dataIndex: 'stageStatus',
        key: 'stageStatus',
        width: 120,
        render: (status: string) => {
          const mapped = getStatusText(status);
          return <Tag color={mapped.color}>{mapped.text}</Tag>;
        },
      },
      {
        title: intl.formatMessage({ id: 'sqlOrder.col.errMsg' }),
        dataIndex: 'errorMessage',
        key: 'errorMessage',
        ellipsis: {
          showTitle: false,
        },
        render: (text: string) => {
          if (!text || text.trim() === '') {
            return <span style={{ color: '#999' }}>-</span>;
          }
          const isLong = text.length > 50;
          return (
            <Tooltip title={isLong ? text : null} placement="topLeft">
              <span style={{ color: text ? '#ff4d4f' : '#999' }}>
                {isLong ? `${text.substring(0, 50)}...` : text}
              </span>
            </Tooltip>
          );
        },
      },
      {
        title: intl.formatMessage({ id: 'sqlOrder.col.rows' }),
        dataIndex: 'affectedRows',
        key: 'affectedRows',
        width: 100,
        render: (rows: number) => rows || 0,
      },
    ],
    [intl, getStatusText]
  );

  // 打开提交弹窗
  const handleOpenSubmit = () => {
    const values = form.getFieldsValue();
    if (!values.instanceId || !values.databaseName) {
      message.warning(intl.formatMessage({ id: 'sqlOrder.editor.warnInstDb' }));
      return;
    }
    if (!sqlContent.trim()) {
      message.warning(intl.formatMessage({ id: 'sqlOrder.editor.enterSql' }));
      return;
    }

    submitForm.setFieldsValue({
      remark: '',
    });
    setSubmitModalVisible(true);
  };

  // 提交工单
  const { loading: submitting, run: handleSubmit } = useRequest(
    async () => {
      const values = form.getFieldsValue();
      const submitValues = await submitForm.validateFields();

      // 处理SQL：去除开头和结尾的空格、换行符等空白字符
      const processedSQL = sqlContent.trim().replace(/^[\s\n\r]+|[\s\n\r]+$/g, '');

      if (!processedSQL) {
        message.warning(intl.formatMessage({ id: 'sqlOrder.editor.sqlEmpty' }));
        return;
      }

      const response = await createSQLOrder({
        instanceId: values.instanceId,
        databaseName: values.databaseName,
        sqlContent: processedSQL,
        orderType: 0, // DDL
        needBackup: true,
        executeMethod: 0,
        remark: submitValues.remark || '',
      });

      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'sqlOrder.create.createOk' }));
        setSubmitModalVisible(false);
        navigate('/sql-order/list');
      } else {
        message.error(response.message || intl.formatMessage({ id: 'sqlOrder.create.createFail' }));
      }
    },
    { manual: true }
  );


  const selectedInstance = instances.find((inst) => inst.id === selectedInstanceId);
  const dbTypeMap: Record<string, string> = {
    mysql: 'MySQL',
    redis: 'Redis',
    clickhouse: 'ClickHouse',
  };

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column', margin: '-20px -18px', padding: '0' }}>
      <div style={{ padding: '16px 16px 8px 16px', borderBottom: '1px solid #f0f0f0', display: 'flex', alignItems: 'center', gap: '8px' }}>
        <span onClick={() => navigate(-1)} style={{ cursor: 'pointer', fontSize: '16px' }}>←</span>
        <h2 style={{ margin: 0, fontSize: '16px', fontWeight: 500 }}>{intl.formatMessage({ id: 'sqlOrder.ddl.title' })}</h2>
      </div>
      <div style={{ flex: 1, padding: '8px', overflow: 'hidden' }}>
        <div style={{ display: 'flex', height: '100%', margin: 0, overflow: 'hidden' }}>
        {/* 左侧：选择器 */}
        <div
          style={{
            width: `${leftPanelWidthPct}%`,
            flex: `0 0 ${leftPanelWidthPct}%`,
            minWidth: 148,
            maxWidth: '32%',
            transition: isLeftPaneDragging ? 'none' : 'width 0.15s ease-out',
            paddingRight: 6,
            height: '100%',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          <Card
            style={{ height: '100%', display: 'flex', flexDirection: 'column', margin: 0, borderRadius: '4px' }}
            bodyStyle={{ flex: 1, display: 'flex', flexDirection: 'column', padding: '6px 8px', overflow: 'auto' }}
          >
            <Form form={form} layout="vertical" size="small" style={{ flex: 1 }}>
              <Form.Item
                name="instanceId"
                label={intl.formatMessage({ id: 'sqlOrder.label.instance' })}
                rules={[{ required: true, message: intl.formatMessage({ id: 'sqlOrder.rule.pickInstance' }) }]}
                style={{ marginBottom: '8px' }}
              >
                <Select
                  placeholder={intl.formatMessage({ id: 'sqlOrder.ph.pickInstance' })}
                  loading={instancesLoading}
                  onChange={handleInstanceChange}
                  showSearch
                  filterOption={(input, option) => {
                    const label = option?.label || option?.children;
                    return String(label || '').toLowerCase().includes(input.toLowerCase());
                  }}
                >
                  {instances.map((inst) => (
                    <Option key={inst.id} value={inst.id} label={`${inst.name} (${inst.host}:${inst.port})`}>
                      <Space size={8}>
                        <DbTypeIcon dbType={inst.dbType} size={18} />
                        <span>
                          {inst.name}{' '}
                          <span style={{ color: '#8c8c8c', fontSize: 12 }}>
                            ({inst.host}:{inst.port})
                          </span>
                        </span>
                      </Space>
                    </Option>
                  ))}
                </Select>
              </Form.Item>

              {/* 实例类型显示 */}
              {selectedInstance && (
                <div style={{ marginBottom: '8px' }}>
                  <Tag color="blue" style={{ margin: 0, fontSize: '12px' }}>
                    {intl.formatMessage({ id: 'sqlOrder.label.instanceType' })}{' '}
                    {dbTypeMap[selectedInstance.dbType] || selectedInstance.dbType.toUpperCase()}
                  </Tag>
                </div>
              )}

              <Form.Item
                name="databaseName"
                label={intl.formatMessage({ id: 'sqlOrder.label.database' })}
                rules={[{ required: true, message: intl.formatMessage({ id: 'sqlOrder.rule.pickDatabase' }) }]}
                style={{ marginBottom: '8px' }}
              >
                <Select
                  placeholder={intl.formatMessage({ id: 'sqlOrder.ph.pickDatabase' })}
                  loading={databasesLoading}
                  disabled={!selectedInstanceId}
                  onChange={handleDatabaseChange}
                  showSearch
                  filterOption={(input, option) => {
                    const label = option?.label || option?.children;
                    return String(label || '').toLowerCase().includes(input.toLowerCase());
                  }}
                >
                  {databases.map((db) => (
                    <Option key={db} value={db} label={db}>
                      <Space size={8}>
                        <DatabaseOutlined style={{ color: '#1677ff', fontSize: 16 }} />
                        <span>{db}</span>
                      </Space>
                    </Option>
                  ))}
                </Select>
              </Form.Item>

              <Form.Item name="tableName" label={intl.formatMessage({ id: 'sqlOrder.label.pickTable' })} style={{ marginBottom: '8px' }}>
                <Select
                  placeholder={intl.formatMessage({ id: 'sqlOrder.ph.pickTable' })}
                  loading={tablesLoading}
                  disabled={!selectedDatabase}
                  onChange={handleTableChange}
                  showSearch
                  filterOption={(input, option) => {
                    const label = option?.label || option?.children;
                    return String(label || '').toLowerCase().includes(input.toLowerCase());
                  }}
                >
                  {tables.map((table) => (
                    <Option key={table} value={table} label={table}>
                      <Space size={8}>
                        <TableOutlined style={{ color: '#52c41a', fontSize: 16 }} />
                        <span>{table}</span>
                      </Space>
                    </Option>
                  ))}
                </Select>
              </Form.Item>

              {/* 操作按钮 */}
              <div style={{ marginTop: '8px' }}>
                <Space direction="vertical" style={{ width: '100%' }} size="small">
                  <Button
                    icon={<FormatPainterOutlined />}
                    onClick={handleBeautify}
                    style={{ width: '100%', backgroundColor: '#f0f0f0', borderColor: '#d9d9d9' }}
                  >
                    {intl.formatMessage({ id: 'sqlOrder.editor.sqlBeautify' })}
                  </Button>
                  <Button
                    icon={<SafetyOutlined />}
                    loading={auditLoading}
                    onClick={handleDDLCheck}
                    style={{ width: '100%', backgroundColor: '#fff7e6', borderColor: '#ffd591', color: '#d46b08' }}
                  >
                    {intl.formatMessage({ id: 'sqlOrder.editor.ddlCheck' })}
                  </Button>
                  <Button
                    type="primary"
                    icon={<SendOutlined />}
                    onClick={handleOpenSubmit}
                    style={{ width: '100%' }}
                  >
                    {intl.formatMessage({ id: 'sqlOrder.create.submit' })}
                  </Button>
                </Space>
              </div>
            </Form>
          </Card>
        </div>

        {/* 拖拽分隔条：左侧选择器 <-> SQL 编辑器 */}
        <div
          className="sql-order-splitter"
          style={{
            backgroundColor: isLeftPaneDragging ? '#1677ff' : undefined,
            flexShrink: 0,
          }}
          onMouseDown={onLeftSplitterMouseDown}
        />

        {/* 中间：SQL编辑器和表DDL */}
        <div style={{ flex: 1, minWidth: 0, paddingLeft: 6, display: 'flex', gap: 0, height: '100%' }}>
          <div
            style={{
              width: selectedTable ? `${sqlEditorWidth}%` : '100%',
              display: 'flex',
              flexDirection: 'column',
              transition: isDragging ? 'none' : 'width 0.2s',
              height: '100%'
            }}
          >
          <Card
            title={<><FileTextOutlined /> {intl.formatMessage({ id: 'sqlOrder.label.sqlStmt' })}</>}
            style={{ height: selectedTable ? 'calc(100% - 180px)' : '100%', display: 'flex', flexDirection: 'column', margin: 0, marginBottom: selectedTable ? 4 : 0, borderRadius: '4px', flex: 1, minHeight: 0 }}
            bodyStyle={{ flex: 1, display: 'flex', flexDirection: 'column', padding: 0, minHeight: 0 }}
          >
            <div style={{ flex: 1, minHeight: 0 }}>
              <Editor
                height="100%"
                defaultLanguage="sql"
                value={sqlContent}
                onChange={(value) => setSqlContent(value || '')}
                theme="vs"
                options={{
                  minimap: { enabled: false },
                  fontSize: 14,
                  wordWrap: 'on',
                  automaticLayout: true,
                }}
              />
            </div>
          </Card>

          {/* DDL检测结果 */}
          {auditResult.length > 0 && (
            <Card title={intl.formatMessage({ id: 'sqlOrder.card.checkResult' })} style={{ marginTop: 4, margin: 0, borderRadius: '4px' }}>
              <Table
                columns={auditColumns}
                dataSource={auditResult.map((record, index) => ({ ...record, key: index }))}
                pagination={false}
                size="small"
                scroll={{ y: 200 }}
              />
            </Card>
          )}
          </div>

          {/* 拖拽分隔条 */}
          {selectedTable && (
            <div
              className="sql-order-splitter"
              style={{
                backgroundColor: isDragging ? '#1677ff' : undefined,
                position: 'relative',
              }}
              onMouseDown={(e) => {
                setIsDragging(true);
                e.preventDefault();
                const startX = e.clientX;
                const startWidth = sqlEditorWidth;
                const container = e.currentTarget.parentElement;
                if (!container) return;

                const handleMouseMove = (moveEvent: MouseEvent) => {
                  const deltaX = moveEvent.clientX - startX;
                  const containerWidth = container.offsetWidth;
                  const deltaPercent = (deltaX / containerWidth) * 100;
                  const newWidth = Math.max(30, Math.min(70, startWidth + deltaPercent));
                  setSqlEditorWidth(newWidth);
                };

                const handleMouseUp = () => {
                  setIsDragging(false);
                  document.removeEventListener('mousemove', handleMouseMove);
                  document.removeEventListener('mouseup', handleMouseUp);
                  document.body.style.cursor = '';
                  document.body.style.userSelect = '';
                };

                document.addEventListener('mousemove', handleMouseMove);
                document.addEventListener('mouseup', handleMouseUp);
                document.body.style.cursor = 'col-resize';
                document.body.style.userSelect = 'none';
              }}
            />
          )}

          {/* 右侧：表DDL（只有选择表时才显示） */}
          {selectedTable && (
            <div
              style={{
                width: `${100 - sqlEditorWidth}%`,
                display: 'flex',
                flexDirection: 'column',
                transition: isDragging ? 'none' : 'width 0.2s',
                height: '100%'
              }}
            >
            <Card
              title={intl.formatMessage({ id: 'sqlOrder.card.tableDdl' })}
              extra={
                <Button
                  type="link"
                  size="small"
                  icon={<ReloadOutlined />}
                  loading={ddlLoading}
                  onClick={() => {
                    if (selectedInstanceId && selectedDatabase && selectedTable) {
                      loadTableDDL({
                        instanceId: selectedInstanceId,
                        database: selectedDatabase,
                        table: selectedTable,
                      });
                    }
                  }}
                >
                  {intl.formatMessage({ id: 'sqlOrder.refresh' })}
                </Button>
              }
              style={{ height: '100%', display: 'flex', flexDirection: 'column', margin: 0, borderRadius: '4px', flex: 1, minHeight: 0 }}
              bodyStyle={{ flex: 1, display: 'flex', flexDirection: 'column', padding: 0, minHeight: 0 }}
            >
              <div style={{ flex: 1, minHeight: 0 }}>
                {ddlLoading ? (
                  <div style={{ textAlign: 'center', padding: '20px' }}>{intl.formatMessage({ id: 'common.loading' })}</div>
                ) : tableDDL ? (
                  <Editor
                    height="100%"
                    defaultLanguage="sql"
                    value={tableDDL}
                    theme="vs"
                    options={{
                      readOnly: true,
                      minimap: { enabled: false },
                      fontSize: 12,
                      wordWrap: 'on',
                      automaticLayout: true,
                    }}
                  />
                ) : (
                  <div style={{ textAlign: 'center', padding: '20px', color: '#999' }}>
                    {intl.formatMessage({ id: 'sqlOrder.pickTableForDdl' })}
                  </div>
                )}
              </div>

            </Card>
            </div>
          )}
        </div>
      </div>
      </div>

      {/* 提交工单弹窗 */}
      <Modal
        title={intl.formatMessage({ id: 'sqlOrder.modal.submit' })}
        open={submitModalVisible}
        onOk={handleSubmit}
        onCancel={() => setSubmitModalVisible(false)}
        confirmLoading={submitting}
        width={700}
      >
        <Form form={submitForm} layout="vertical">
          <Form.Item label={intl.formatMessage({ id: 'sqlOrder.label.instance' })}>
            <Input value={selectedInstance?.name || ''} disabled />
          </Form.Item>
          <Form.Item label={intl.formatMessage({ id: 'sqlOrder.label.database' })}>
            <Input value={selectedDatabase} disabled />
          </Form.Item>
          {selectedTable && (
            <Form.Item label={intl.formatMessage({ id: 'sqlOrder.modal.table' })}>
              <Input value={selectedTable} disabled />
            </Form.Item>
          )}
          <Form.Item label={intl.formatMessage({ id: 'sqlOrder.label.sqlStmt' })}>
            <TextArea
              value={sqlContent}
              disabled
              rows={8}
              style={{ fontFamily: 'monospace' }}
            />
          </Form.Item>
          <Form.Item
            name="remark"
            label={intl.formatMessage({ id: 'common.remark' })}
            rules={[{ max: 500, message: intl.formatMessage({ id: 'sqlOrder.remarkMax' }) }]}
          >
            <TextArea
              placeholder={intl.formatMessage({ id: 'sqlOrder.create.remarkPh' })}
              rows={3}
              maxLength={500}
              showCount
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};
