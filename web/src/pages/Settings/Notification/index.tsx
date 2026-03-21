import { PageContainer } from '@ant-design/pro-components';
import {
  Card,
  Form,
  Input,
  Switch,
  Button,
  message,
  Space,
  Tabs,
  Typography,
  Alert,
} from 'antd';
import {
  WechatOutlined,
  SendOutlined,
  SaveOutlined,
  BellOutlined,
} from '@ant-design/icons';
import { useEffect, useState, useMemo } from 'react';
import {
  getNotificationConfigs,
  updateNotificationConfig,
  testNotificationChannel,
} from '@/services/api';
import { useIntl } from '@/locales';

const { Text, Link } = Typography;

interface ChannelConfig {
  enabled: boolean;
  webhook_url: string;
  extra_config: string;
}

const defaultConfig: ChannelConfig = { enabled: false, webhook_url: '', extra_config: '' };

const NotificationSettings: React.FC = () => {
  const intl = useIntl();
  const [configs, setConfigs] = useState<Record<string, ChannelConfig>>({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState<Record<string, boolean>>({});
  const [testing, setTesting] = useState<Record<string, boolean>>({});

  const CHANNELS = useMemo(
    () => [
      {
        key: 'wechatwork',
        label: intl.formatMessage({ id: 'notify.wxwork' }),
        icon: <WechatOutlined />,
        webhookPlaceholder: 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...',
        helpText: (
          <>
            {intl.formatMessage({ id: 'notify.help.wxwork' })}
            <Link
              href="https://developer.work.weixin.qq.com/document/path/91770"
              target="_blank"
              style={{ marginLeft: 4 }}
            >
              {intl.formatMessage({ id: 'notify.docLink' })}
            </Link>
          </>
        ),
        extraFields: null as React.ReactNode,
      },
      {
        key: 'feishu',
        label: intl.formatMessage({ id: 'notify.feishu' }),
        icon: <BellOutlined />,
        webhookPlaceholder: 'https://open.feishu.cn/open-apis/bot/v2/hook/...',
        helpText: (
          <>
            {intl.formatMessage({ id: 'notify.help.feishu' })}
            <Link
              href="https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot"
              target="_blank"
              style={{ marginLeft: 4 }}
            >
              {intl.formatMessage({ id: 'notify.docLink' })}
            </Link>
          </>
        ),
        extraFields: null as React.ReactNode,
      },
      {
        key: 'dingtalk',
        label: intl.formatMessage({ id: 'notify.dingtalk' }),
        icon: <SendOutlined />,
        webhookPlaceholder: 'https://oapi.dingtalk.com/robot/send?access_token=...',
        helpText: (
          <>
            {intl.formatMessage({ id: 'notify.help.dingtalk' })}
            <Link
              href="https://open.dingtalk.com/document/robots/custom-robot-access"
              target="_blank"
              style={{ marginLeft: 4 }}
            >
              {intl.formatMessage({ id: 'notify.docLink' })}
            </Link>
          </>
        ),
        extraFields: (
          <Form.Item
            name="phones"
            label={intl.formatMessage({ id: 'notify.atMobiles' })}
            extra={intl.formatMessage({ id: 'notify.atMobilesExtra' })}
          >
            <Input placeholder="13800138000,13900139000" />
          </Form.Item>
        ),
      },
    ],
    [intl]
  );

  useEffect(() => {
    loadConfigs();
  }, []);

  const loadConfigs = async () => {
    setLoading(true);
    try {
      const res = await getNotificationConfigs();
      if (res.code === 200 && Array.isArray(res.data)) {
        const map: Record<string, ChannelConfig> = {};
        for (const item of res.data) {
          map[item.channel] = {
            enabled: item.enabled,
            webhook_url: item.webhook_url || '',
            extra_config: item.extra_config || '',
          };
        }
        setConfigs(map);
      }
    } catch {
    } finally {
      setLoading(false);
    }
  };

  const getFormForChannel = (channelKey: string) => {
    return configs[channelKey] ?? defaultConfig;
  };

  const handleSave = async (channelKey: string, values: any) => {
    let extraConfig = '';
    if (channelKey === 'dingtalk' && values.phones) {
      const phones = values.phones
        .split(',')
        .map((p: string) => p.trim())
        .filter(Boolean);
      extraConfig = JSON.stringify({ phones });
    }

    setSaving((prev) => ({ ...prev, [channelKey]: true }));
    try {
      const res = await updateNotificationConfig(channelKey, {
        enabled: values.enabled ?? false,
        webhook_url: values.webhook_url || '',
        extra_config: extraConfig,
      });
      if (res.code === 200) {
        message.success(intl.formatMessage({ id: 'notify.saveOk' }));
        setConfigs((prev) => ({
          ...prev,
          [channelKey]: {
            enabled: values.enabled ?? false,
            webhook_url: values.webhook_url || '',
            extra_config: extraConfig,
          },
        }));
      } else {
        message.error(res.message || intl.formatMessage({ id: 'notify.saveFail' }));
      }
    } catch (e: any) {
      message.error(e.message || intl.formatMessage({ id: 'notify.saveFail' }));
    } finally {
      setSaving((prev) => ({ ...prev, [channelKey]: false }));
    }
  };

  const handleTest = async (channelKey: string) => {
    setTesting((prev) => ({ ...prev, [channelKey]: true }));
    try {
      const res = await testNotificationChannel(channelKey);
      if (res.code === 200) {
        message.success(intl.formatMessage({ id: 'notify.testOk' }));
      } else {
        message.error(res.message || intl.formatMessage({ id: 'notify.testFail' }));
      }
    } catch (e: any) {
      message.error(e.message || intl.formatMessage({ id: 'notify.testFail' }));
    } finally {
      setTesting((prev) => ({ ...prev, [channelKey]: false }));
    }
  };

  const tabItems = CHANNELS.map((channel) => {
    const cfg = getFormForChannel(channel.key);

    let initialPhones = '';
    if (channel.key === 'dingtalk' && cfg.extra_config) {
      try {
        const parsed = JSON.parse(cfg.extra_config);
        if (Array.isArray(parsed.phones)) {
          initialPhones = parsed.phones.join(',');
        }
      } catch {
      }
    }

    return {
      key: channel.key,
      label: (
        <span>
          {channel.icon} {channel.label}
        </span>
      ),
      children: (
        <Card loading={loading}>
          <Alert
            message={channel.helpText}
            type="info"
            showIcon
            style={{ marginBottom: 24 }}
          />
          <Form
            layout="vertical"
            initialValues={{
              enabled: cfg.enabled,
              webhook_url: cfg.webhook_url,
              phones: initialPhones,
            }}
            key={`${channel.key}-${cfg.webhook_url}`}
            onFinish={(values) => handleSave(channel.key, values)}
          >
            <Form.Item name="enabled" label={intl.formatMessage({ id: 'notify.enable' })} valuePropName="checked">
              <Switch />
            </Form.Item>

            <Form.Item
              name="webhook_url"
              label={intl.formatMessage({ id: 'notify.webhook' })}
              rules={[{ required: false }]}
            >
              <Input
                placeholder={channel.webhookPlaceholder}
                style={{ fontFamily: 'monospace' }}
              />
            </Form.Item>

            {channel.extraFields}

            <Form.Item>
              <Space>
                <Button
                  type="primary"
                  htmlType="submit"
                  icon={<SaveOutlined />}
                  loading={saving[channel.key]}
                >
                  {intl.formatMessage({ id: 'notify.saveConfig' })}
                </Button>
                <Button
                  icon={<SendOutlined />}
                  loading={testing[channel.key]}
                  disabled={!cfg.webhook_url}
                  onClick={() => handleTest(channel.key)}
                >
                  {intl.formatMessage({ id: 'notify.sendTest' })}
                </Button>
              </Space>
            </Form.Item>
          </Form>
        </Card>
      ),
    };
  });

  return (
    <PageContainer title={intl.formatMessage({ id: 'pageTitle.notification' })}>
      <Card>
        <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
          {intl.formatMessage({ id: 'notify.intro' })}
        </Text>
        <Tabs items={tabItems} />
      </Card>
    </PageContainer>
  );
};

export default NotificationSettings;
