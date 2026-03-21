import { LockOutlined, MailOutlined, UserOutlined, PhoneOutlined } from '@ant-design/icons';
import { ProFormText, ProForm } from '@ant-design/pro-components';
import { useNavigate } from 'react-router-dom';
import { Button, Card, message } from 'antd';
import { useIntl } from '@/locales';
import { register } from '@/services/api';
import LanguageSwitch from '@/components/LanguageSwitch';
import DocumentLang from '@/locales/DocumentLang';
import './index.less';

export default () => {
  const navigate = useNavigate();
  const intl = useIntl();

  const handleSubmit = async (values: API.RegisterRequest) => {
    try {
      const response = await register(values);
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'register.success' }));
        navigate('/login');
      } else {
        message.error(response.message || intl.formatMessage({ id: 'register.failed' }));
      }
    } catch {
      /* ignore */
    }
  };

  return (
    <>
      <DocumentLang />
      <div className="register-container">
      <div
        style={{
          position: 'absolute',
          top: 24,
          right: 24,
          zIndex: 10,
        }}
      >
        <LanguageSwitch />
      </div>
      <Card className="register-card">
        <div className="register-header">
          <h1>{intl.formatMessage({ id: 'app.name' })}</h1>
          <p>{intl.formatMessage({ id: 'register.subtitle' })}</p>
        </div>
        <ProForm
          onFinish={handleSubmit}
          submitter={{
            render: () => (
              <div style={{ textAlign: 'center', marginTop: 24 }}>
                <Button type="primary" htmlType="submit" size="large" block>
                  {intl.formatMessage({ id: 'register.submit' })}
                </Button>
                <div style={{ marginTop: 16 }}>
                  <a href="/#/login">{intl.formatMessage({ id: 'register.loginHint' })}</a>
                </div>
              </div>
            ),
          }}
        >
          <ProFormText
            name="username"
            fieldProps={{
              size: 'large',
              prefix: <UserOutlined />,
            }}
            placeholder={intl.formatMessage({ id: 'login.username' })}
            rules={[
              { required: true, message: intl.formatMessage({ id: 'login.usernameRequired' }) },
              { min: 3, message: intl.formatMessage({ id: 'register.usernameMin' }) },
            ]}
          />
          <ProFormText.Password
            name="password"
            fieldProps={{
              size: 'large',
              prefix: <LockOutlined />,
            }}
            placeholder={intl.formatMessage({ id: 'login.password' })}
            rules={[
              { required: true, message: intl.formatMessage({ id: 'login.passwordRequired' }) },
              { min: 6, message: intl.formatMessage({ id: 'register.passwordMin' }) },
            ]}
          />
          <ProFormText
            name="email"
            fieldProps={{
              size: 'large',
              prefix: <MailOutlined />,
            }}
            placeholder={intl.formatMessage({ id: 'register.email' })}
            rules={[
              { required: true, message: intl.formatMessage({ id: 'register.emailRequired' }) },
              { type: 'email', message: intl.formatMessage({ id: 'register.emailInvalid' }) },
            ]}
          />
          <ProFormText
            name="realName"
            fieldProps={{
              size: 'large',
            }}
            placeholder={intl.formatMessage({ id: 'register.realName' })}
            rules={[{ required: true, message: intl.formatMessage({ id: 'register.realNameRequired' }) }]}
          />
          <ProFormText
            name="phone"
            fieldProps={{
              size: 'large',
              prefix: <PhoneOutlined />,
            }}
            placeholder={intl.formatMessage({ id: 'register.phoneOptional' })}
          />
        </ProForm>
      </Card>
    </div>
    </>
  );
};
