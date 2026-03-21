import { PageContainer, ProForm, ProFormSelect, ProFormText } from '@ant-design/pro-components';
import { Card, Button, message, Space } from 'antd';
import { PlayCircleOutlined } from '@ant-design/icons';
import { redisQuery, listInstances } from '@/services/api';
import { useRequest } from 'ahooks';
import { useState } from 'react';
import { useIntl } from '@/locales';

export default () => {
  const intl = useIntl();
  const [command, setCommand] = useState('');
  const [queryResult, setQueryResult] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const { data: instancesData } = useRequest(() =>
    listInstances({ page: 1, pageSize: 1000, dbType: 'redis', instancePermissionScope: 'query' })
  );
  const instances = instancesData?.list || [];

  const handleQuery = async (values: { instanceId: string }) => {
    if (!command.trim()) {
      message.warning(intl.formatMessage({ id: 'queryEngine.warnRedisCmd' }));
      return;
    }

    setLoading(true);
    try {
      const response = await redisQuery({
        instanceId: values.instanceId,
        command: command.trim(),
      });
      if (response.code === 200 && response.data) {
        setQueryResult(response.data);
        message.success(intl.formatMessage({ id: 'queryEngine.redisOk' }));
      } else {
        message.error(response.message || intl.formatMessage({ id: 'queryEngine.fail' }));
      }
    } catch {
    } finally {
      setLoading(false);
    }
  };

  return (
    <PageContainer title={intl.formatMessage({ id: 'queryEngine.title.redis' })}>
      <Card>
        <ProForm
          onFinish={handleQuery}
          submitter={{
            render: () => (
              <Button type="primary" htmlType="submit" icon={<PlayCircleOutlined />} loading={loading}>
                {intl.formatMessage({ id: 'queryEngine.runCmd' })}
              </Button>
            ),
          }}
        >
          <ProFormSelect
            name="instanceId"
            label={intl.formatMessage({ id: 'queryEngine.col.redisInst' })}
            placeholder={intl.formatMessage({ id: 'queryEngine.ph.redisInst' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'queryEngine.rule.redisInst' }) }]}
            options={instances.map((inst) => ({
              label: `${inst.name} (${inst.host}:${inst.port})`,
              value: inst.id,
            }))}
            fieldProps={{
              showSearch: true,
            }}
          />
          <ProFormText
            name="command"
            label={intl.formatMessage({ id: 'queryEngine.col.redisCmd' })}
            placeholder={intl.formatMessage({ id: 'queryEngine.ph.redisCmd' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'queryEngine.rule.redisCmd' }) }]}
            fieldProps={{
              value: command,
              onChange: (e) => setCommand(e.target.value),
            }}
          />
        </ProForm>
      </Card>

      {queryResult && (
        <Card title={intl.formatMessage({ id: 'queryEngine.result' })} style={{ marginTop: 16 }}>
          <pre
            style={{
              background: '#f5f5f5',
              padding: '12px',
              borderRadius: '4px',
              overflow: 'auto',
            }}
          >
            {typeof queryResult === 'string' ? queryResult : JSON.stringify(queryResult, null, 2)}
          </pre>
        </Card>
      )}

      <Card title={intl.formatMessage({ id: 'queryEngine.redisCheatTitle' })} style={{ marginTop: 16 }}>
        <Space direction="vertical" style={{ width: '100%' }}>
          <div>
            <code>GET key</code> — {intl.formatMessage({ id: 'queryEngine.redisCheat.get' })}
          </div>
          <div>
            <code>SET key value</code> — {intl.formatMessage({ id: 'queryEngine.redisCheat.set' })}
          </div>
          <div>
            <code>KEYS pattern</code> — {intl.formatMessage({ id: 'queryEngine.redisCheat.keys' })}
          </div>
          <div>
            <code>INFO</code> — {intl.formatMessage({ id: 'queryEngine.redisCheat.info' })}
          </div>
          <div>
            <code>DBSIZE</code> — {intl.formatMessage({ id: 'queryEngine.redisCheat.dbsize' })}
          </div>
        </Space>
      </Card>
    </PageContainer>
  );
};
