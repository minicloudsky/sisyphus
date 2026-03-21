import { PageContainer, ProForm, ProFormTextArea } from '@ant-design/pro-components';
import { Card, Button, message, Space } from 'antd';
import { PlayCircleOutlined, CopyOutlined } from '@ant-design/icons';
import { generateMybatisXML } from '@/services/api';
import { useRequest } from 'ahooks';
import { useState } from 'react';
import Editor from '@monaco-editor/react';
import { useIntl } from '@/locales';

export default () => {
  const intl = useIntl();
  const [ddl, setDdl] = useState('');
  const [result, setResult] = useState('');
  const [loading, setLoading] = useState(false);

  const { run: handleGenerate } = useRequest(
    async () => {
      if (!ddl.trim()) {
        message.warning(intl.formatMessage({ id: 'tools.mybatis.warnDdl' }));
        return;
      }

      setLoading(true);
      try {
        const response = await generateMybatisXML({ ddl: ddl.trim() });
        if (response.code === 200 && response.data) {
          setResult(response.data.xml || '');
          message.success(intl.formatMessage({ id: 'tools.mybatis.genOk' }));
        } else {
          message.error(response.message || intl.formatMessage({ id: 'tools.mybatis.genFail' }));
        }
      } catch {
      } finally {
        setLoading(false);
      }
    },
    { manual: true }
  );

  const handleCopy = () => {
    if (result) {
      navigator.clipboard.writeText(result);
      message.success(intl.formatMessage({ id: 'tools.mybatis.copyOk' }));
    }
  };

  return (
    <PageContainer title={intl.formatMessage({ id: 'tools.mybatis.title' })}>
      <Card>
        <ProForm
          onFinish={async () => { await handleGenerate(); }}
          submitter={{
            render: () => (
              <Space>
                <Button type="primary" htmlType="submit" icon={<PlayCircleOutlined />} loading={loading}>
                  {intl.formatMessage({ id: 'tools.mybatis.gen' })}
                </Button>
                {result && (
                  <Button icon={<CopyOutlined />} onClick={handleCopy}>
                    {intl.formatMessage({ id: 'tools.mybatis.copy' })}
                  </Button>
                )}
              </Space>
            ),
          }}
        >
          <ProForm.Item
            name="ddl"
            label={intl.formatMessage({ id: 'tools.mybatis.label.ddl' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'tools.mybatis.rule.ddl' }) }]}
          >
            <Editor
              height="300px"
              defaultLanguage="sql"
              value={ddl}
              onChange={(value) => setDdl(value || '')}
              theme="vs"
              options={{
                minimap: { enabled: false },
                fontSize: 14,
              }}
            />
          </ProForm.Item>
        </ProForm>
      </Card>

      {result && (
        <Card title={intl.formatMessage({ id: 'tools.mybatis.card.out' })} style={{ marginTop: 16 }}>
          <Editor
            height="500px"
            defaultLanguage="xml"
            value={result}
            theme="vs"
            options={{
              readOnly: true,
              minimap: { enabled: false },
            }}
          />
        </Card>
      )}

      <Card title={intl.formatMessage({ id: 'tools.mybatis.card.help' })} style={{ marginTop: 16 }}>
        <ul>
          <li>{intl.formatMessage({ id: 'tools.mybatis.help1' })}</li>
          <li>{intl.formatMessage({ id: 'tools.mybatis.help2' })}</li>
          <li>{intl.formatMessage({ id: 'tools.mybatis.help3' })}</li>
          <li>{intl.formatMessage({ id: 'tools.mybatis.help4' })}</li>
        </ul>
      </Card>
    </PageContainer>
  );
};
