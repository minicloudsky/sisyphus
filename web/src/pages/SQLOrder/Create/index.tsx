import { PageContainer, ProForm, ProFormSelect, ProFormTextArea, ProFormText } from '@ant-design/pro-components';
import { message, Card, Tag } from 'antd';
import { useNavigate } from 'react-router-dom';
import { createSQLOrder, auditSQL, checkSQLType, listInstances } from '@/services/api';
import { useRequest } from 'ahooks';
import Editor from '@monaco-editor/react';
import { useState } from 'react';
import { useIntl } from '@/locales';
import './index.less';

export default () => {
  const intl = useIntl();
  const navigate = useNavigate();
  const [sqlContent, setSqlContent] = useState('');
  const [auditResult, setAuditResult] = useState<API.SQLReviewRecord[]>([]);

  const { data: instancesData } = useRequest(() => listInstances({ page: 1, pageSize: 1000 }));
  const instances = instancesData?.list || [];

  const { run: handleAudit } = useRequest(
    async (instanceId: string, databaseName: string, sql: string) => {
      const response = await auditSQL({ instanceId, databaseName, sqlContent: sql });
      if (response.code === 200) {
        setAuditResult(response.data?.records || []);
        message.success(intl.formatMessage({ id: 'sqlOrder.create.auditOk' }));
      } else {
        message.error(response.message || intl.formatMessage({ id: 'sqlOrder.create.auditFail' }));
      }
    },
    { manual: true }
  );

  const { run: handleCheckType } = useRequest(
    async (sql: string) => {
      const response = await checkSQLType({ sqlContent: sql });
      if (response.code === 200 && response.data) {
        message.info(
          intl.formatMessage(
            { id: 'sqlOrder.create.typeInfo' },
            {
              type: response.data.sqlType,
              self: response.data.canSelfExecute ? intl.formatMessage({ id: 'common.yes' }) : intl.formatMessage({ id: 'common.no' }),
            }
          )
        );
      }
    },
    { manual: true }
  );

  const handleSubmit = async (values: API.CreateOrderRequest) => {
    try {
      const response = await createSQLOrder({
        ...values,
        sqlContent,
      });
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'sqlOrder.create.createOk' }));
        navigate('/sql-order/list');
      } else {
        message.error(response.message || intl.formatMessage({ id: 'sqlOrder.create.createFail' }));
      }
    } catch (error: any) {
      if (error?.errorFields) return;
    }
  };

  return (
    <PageContainer title={intl.formatMessage({ id: 'sqlOrder.create.pageTitle' })} onBack={() => navigate(-1)}>
      <Card>
        <ProForm
          onFinish={handleSubmit}
          initialValues={{
            orderType: 0,
            needBackup: true,
            executeMethod: 0,
          }}
          submitter={{
            searchConfig: {
              submitText: intl.formatMessage({ id: 'sqlOrder.create.submit' }),
            },
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
              filterOption: (input, option) =>
                String(option?.label ?? '').toLowerCase().includes(input.toLowerCase()),
            }}
          />
          <ProFormText
            name="databaseName"
            label={intl.formatMessage({ id: 'sqlOrder.create.dbName' })}
            placeholder={intl.formatMessage({ id: 'sqlOrder.create.dbNamePh' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'sqlOrder.create.dbNameRule' }) }]}
          />
          <ProFormSelect
            name="orderType"
            label={intl.formatMessage({ id: 'sqlOrder.create.orderType' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'sqlOrder.create.orderTypeRule' }) }]}
            options={[
              { label: 'DDL', value: 0 },
              { label: 'DML', value: 1 },
              { label: intl.formatMessage({ id: 'sqlOrder.create.export' }), value: 2 },
            ]}
          />
          <ProForm.Item
            name="sqlContent"
            label={intl.formatMessage({ id: 'sqlOrder.create.sqlContent' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'sqlOrder.create.sqlRule' }) }]}
          >
            <div className="sql-editor-container">
              <Editor
                height="300px"
                defaultLanguage="sql"
                value={sqlContent}
                onChange={(value) => setSqlContent(value || '')}
                theme="vs"
                options={{
                  minimap: { enabled: false },
                  fontSize: 14,
                  lineNumbers: 'on',
                  scrollBeyondLastLine: false,
                }}
              />
              <div className="sql-editor-actions">
                <a onClick={() => handleCheckType(sqlContent)}>{intl.formatMessage({ id: 'sqlOrder.create.checkType' })}</a>
                <a onClick={() => {
                  const instanceId = document.querySelector('[name="instanceId"]')?.getAttribute('value');
                  const databaseName = document.querySelector('[name="databaseName"]')?.getAttribute('value');
                  if (instanceId && databaseName && sqlContent) {
                    handleAudit(instanceId, databaseName, sqlContent);
                  } else {
                    message.warning(intl.formatMessage({ id: 'sqlOrder.create.warnPickInst' }));
                  }
                }}>{intl.formatMessage({ id: 'sqlOrder.create.sqlAudit' })}</a>
              </div>
            </div>
          </ProForm.Item>
          {auditResult.length > 0 && (
            <ProForm.Item label={intl.formatMessage({ id: 'sqlOrder.create.auditResult' })}>
              <div className="audit-result">
                {auditResult.map((record, index) => (
                  <div key={index} className={`audit-item audit-level-${record.errorLevel}`}>
                    <div className="audit-header">
                      <span>{intl.formatMessage({ id: 'sqlOrder.create.sqlSeq' }, { n: record.sequence })}</span>
                      <Tag color={record.errorLevel === 0 ? 'green' : record.errorLevel === 1 ? 'orange' : 'red'}>
                        {record.errorLevel === 0
                          ? intl.formatMessage({ id: 'sqlOrder.create.pass' })
                          : record.errorLevel === 1
                            ? intl.formatMessage({ id: 'sqlOrder.create.warn' })
                            : intl.formatMessage({ id: 'sqlOrder.create.err' })}
                      </Tag>
                    </div>
                    {record.errorMessage && (
                      <div className="audit-message">{record.errorMessage}</div>
                    )}
                    <div className="audit-sql">{record.sqlText}</div>
                  </div>
                ))}
              </div>
            </ProForm.Item>
          )}
          <ProFormTextArea
            name="remark"
            label={intl.formatMessage({ id: 'sqlOrder.remark' })}
            placeholder={intl.formatMessage({ id: 'sqlOrder.create.remarkPh' })}
            fieldProps={{
              rows: 3,
            }}
          />
          <ProFormSelect
            name="needBackup"
            label={intl.formatMessage({ id: 'sqlOrder.create.needBackup' })}
            options={[
              { label: intl.formatMessage({ id: 'common.yes' }), value: 1 },
              { label: intl.formatMessage({ id: 'common.no' }), value: 0 },
            ]}
          />
          <ProFormSelect
            name="executeMethod"
            label={intl.formatMessage({ id: 'sqlOrder.create.execMethod' })}
            options={[
              { label: intl.formatMessage({ id: 'sqlOrder.exec.auto' }), value: 0 },
              { label: intl.formatMessage({ id: 'sqlOrder.create.ghOst' }), value: 1 },
              { label: intl.formatMessage({ id: 'sqlOrder.create.goInception' }), value: 2 },
              { label: intl.formatMessage({ id: 'sqlOrder.create.ckcheck' }), value: 3 },
            ]}
            extra={intl.formatMessage({ id: 'sqlOrder.create.execExtra' })}
          />
        </ProForm>
      </Card>
    </PageContainer>
  );
};
