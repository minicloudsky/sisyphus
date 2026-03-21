import { LockOutlined, UserOutlined } from '@ant-design/icons';
import { LoginForm, ProFormText } from '@ant-design/pro-components';
import { useNavigate } from 'react-router-dom';
import { message } from 'antd';
import { useIntl } from '@/locales';
import { login as loginService } from '@/services/api';
import { setToken, setUser } from '@/utils/auth';
import LanguageSwitch from '@/components/LanguageSwitch';
import DocumentLang from '@/locales/DocumentLang';
import './index.less';

export default () => {
  const navigate = useNavigate();
  const intl = useIntl();

  const handleSubmit = async (values: API.LoginRequest) => {
    try {
      const response = await loginService(values);
      if (response.code === 200 && response.data) {
        setToken(response.data.token);
        setUser(response.data.user);
        message.success(intl.formatMessage({ id: 'login.success' }));
        navigate('/home');
      } else {
        message.error(response.message || intl.formatMessage({ id: 'login.failed' }));
      }
    } catch {
      /* network */
    }
  };

  return (
    <>
      <DocumentLang />
      <div className="login-container">
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
      <LoginForm
        title={intl.formatMessage({ id: 'app.name' })}
        subTitle={intl.formatMessage({ id: 'login.subtitle' })}
        onFinish={handleSubmit}
        logo={<img alt="logo" src="/logo.svg" />}
      >
        <ProFormText
          name="username"
          fieldProps={{
            size: 'large',
            prefix: <UserOutlined />,
          }}
          placeholder={intl.formatMessage({ id: 'login.username' })}
          rules={[
            {
              required: true,
              message: intl.formatMessage({ id: 'login.usernameRequired' }),
            },
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
            {
              required: true,
              message: intl.formatMessage({ id: 'login.passwordRequired' }),
            },
          ]}
        />
        <div style={{ marginBottom: 24, textAlign: 'right' }}>
          <a href="/#/register">{intl.formatMessage({ id: 'login.registerHint' })}</a>
        </div>
      </LoginForm>
    </div>
    </>
  );
};
