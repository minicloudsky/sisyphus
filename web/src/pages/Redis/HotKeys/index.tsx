import { PageContainer, ProForm, ProFormSelect, ProFormDigit } from '@ant-design/pro-components';
import { Card, Button, message } from 'antd';
import { PlayCircleOutlined } from '@ant-design/icons';
import { analyzeHotKeys, listInstances } from '@/services/api';
import { useRequest } from 'ahooks';
import { useNavigate } from 'react-router-dom';
import { useIntl } from '@/locales';

export default () => {
  const intl = useIntl();
  const navigate = useNavigate();
  const { data: instancesData } = useRequest(() =>
    listInstances({ page: 1, pageSize: 1000, dbType: 'redis', instancePermissionScope: 'query' }),
  );
  const instances = instancesData?.list || [];

  const { run: handleAnalyze, loading } = useRequest(
    async (values: API.AnalyzeHotKeysRequest) => {
      const response = await analyzeHotKeys(values);
      if (response.code === 200 && response.data) {
        message.success(response.data.message || intl.formatMessage({ id: 'redis.submitOk' }));
        navigate('/redis/tasks');
        return true;
      } else {
        message.error(response.message || intl.formatMessage({ id: 'redis.submitFail' }));
        return false;
      }
    },
    { manual: true }
  );

  return (
    <PageContainer title={intl.formatMessage({ id: 'redis.hot.title' })}>
      <Card>
        <ProForm
          onFinish={async (values) => { await handleAnalyze(values as API.AnalyzeHotKeysRequest); }}
          submitter={{
            searchConfig: {
              submitText: intl.formatMessage({ id: 'redis.startAnalyze' }),
            },
            render: () => (
              <Button type="primary" htmlType="submit" icon={<PlayCircleOutlined />} loading={loading}>
                {intl.formatMessage({ id: 'redis.startAnalyze' })}
              </Button>
            ),
          }}
        >
          <ProFormSelect
            name="instanceId"
            label={intl.formatMessage({ id: 'redis.label.inst' })}
            placeholder={intl.formatMessage({ id: 'redis.ph.inst' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'redis.rule.inst' }) }]}
            options={instances.map((inst) => ({
              label: `${inst.name} (${inst.host}:${inst.port})`,
              value: inst.id,
            }))}
            fieldProps={{
              showSearch: true,
            }}
          />
          <ProFormDigit
            name="timeWindowSeconds"
            label={intl.formatMessage({ id: 'redis.label.window' })}
            placeholder={intl.formatMessage({ id: 'redis.ph.window' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'redis.rule.window' }) }]}
            fieldProps={{
              min: 1,
              max: 3600,
            }}
            extra={intl.formatMessage({ id: 'redis.extra.window' })}
          />
        </ProForm>
      </Card>

      <Card title={intl.formatMessage({ id: 'redis.card.help' })} style={{ marginTop: 16 }}>
        <ul>
          <li>{intl.formatMessage({ id: 'redis.help.hot1' })}</li>
          <li>{intl.formatMessage({ id: 'redis.help.hot2' })}</li>
          <li>{intl.formatMessage({ id: 'redis.help.hot3' })}</li>
        </ul>
      </Card>
    </PageContainer>
  );
};
