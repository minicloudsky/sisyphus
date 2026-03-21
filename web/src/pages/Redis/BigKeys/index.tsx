import { PageContainer, ProForm, ProFormSelect, ProFormDigit } from '@ant-design/pro-components';
import { Card, Button, message } from 'antd';
import { PlayCircleOutlined } from '@ant-design/icons';
import { analyzeBigKeys, listInstances } from '@/services/api';
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
    async (values: API.AnalyzeBigKeysRequest) => {
      const response = await analyzeBigKeys(values);
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
    <PageContainer title={intl.formatMessage({ id: 'redis.big.title' })}>
      <Card>
        <ProForm
          onFinish={async (values) => { await handleAnalyze(values as API.AnalyzeBigKeysRequest); }}
          submitter={{
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
            name="sizeThreshold"
            label={intl.formatMessage({ id: 'redis.label.threshold' })}
            placeholder={intl.formatMessage({ id: 'redis.ph.threshold' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'redis.rule.threshold' }) }]}
            fieldProps={{
              min: 1,
            }}
            extra={intl.formatMessage({ id: 'redis.extra.threshold' })}
          />
        </ProForm>
      </Card>

      <Card title={intl.formatMessage({ id: 'redis.card.help' })} style={{ marginTop: 16 }}>
        <ul>
          <li>{intl.formatMessage({ id: 'redis.help.big1' })}</li>
          <li>{intl.formatMessage({ id: 'redis.help.big2' })}</li>
          <li>{intl.formatMessage({ id: 'redis.help.big3' })}</li>
        </ul>
      </Card>
    </PageContainer>
  );
};
