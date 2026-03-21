import { PageContainer } from '@ant-design/pro-components';
import {
  Card,
  Form,
  Input,
  Button,
  message,
  Space,
  Avatar,
  Divider,
  Row,
  Col,
  Tabs,
  Typography,
} from 'antd';
import { UserOutlined, EditOutlined, SaveOutlined, CloseOutlined, LockOutlined, MailOutlined, IdcardOutlined } from '@ant-design/icons';
import { getUser, setUser } from '@/utils/auth';
import { useState } from 'react';
import { updateUser } from '@/services/api';
import { formatDateTime } from '@/utils/format';
import { useIntl } from '@/locales';
import './index.less';

const { Text } = Typography;

export default () => {
  const intl = useIntl();
  const currentUser = getUser();
  const [infoForm] = Form.useForm();
  const [pwdForm] = Form.useForm();
  const [editing, setEditing] = useState(false);
  const [loading, setLoading] = useState(false);
  const [pwdLoading, setPwdLoading] = useState(false);

  const handleEdit = () => {
    setEditing(true);
    infoForm.setFieldsValue(currentUser || {});
  };

  const handleCancel = () => {
    setEditing(false);
    infoForm.resetFields();
  };

  const handleSave = async () => {
    try {
      const values = await infoForm.validateFields();
      setLoading(true);

      const response = await updateUser(values);
      if (response.code === 200 && response.data) {
        setUser(response.data);
        message.success(intl.formatMessage({ id: 'profile.updateOk' }));
        setEditing(false);
      } else {
        message.error(response.message || intl.formatMessage({ id: 'profile.updateFail' }));
      }
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error(error.message || intl.formatMessage({ id: 'profile.updateFail' }));
    } finally {
      setLoading(false);
    }
  };

  const handlePasswordChange = async () => {
    try {
      const values = await pwdForm.validateFields();
      if (values.newPassword !== values.confirmPassword) {
        message.error(intl.formatMessage({ id: 'profile.pwdMismatchForm' }));
        return;
      }
      setPwdLoading(true);
      const response = await updateUser({
        oldPassword: values.oldPassword,
        password: values.newPassword,
      } as any);
      if (response.code === 200) {
        message.success(intl.formatMessage({ id: 'profile.pwdOk' }));
        pwdForm.resetFields();
      } else {
        message.error(response.message || intl.formatMessage({ id: 'profile.pwdFail' }));
      }
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error(error.message || intl.formatMessage({ id: 'profile.pwdFail' }));
    } finally {
      setPwdLoading(false);
    }
  };

  const infoTabContent = (
    <div className="profile-tab-inner">
      <Card
        className="profile-section-card"
        title={intl.formatMessage({ id: 'profile.basic' })}
        extra={
          !editing ? (
            <Button type="primary" icon={<EditOutlined />} onClick={handleEdit}>
              {intl.formatMessage({ id: 'profile.editProfile' })}
            </Button>
          ) : (
            <Space>
              <Button onClick={handleCancel} icon={<CloseOutlined />}>
                {intl.formatMessage({ id: 'common.cancel' })}
              </Button>
              <Button type="primary" loading={loading} onClick={handleSave} icon={<SaveOutlined />}>
                {intl.formatMessage({ id: 'common.save' })}
              </Button>
            </Space>
          )
        }
      >
        <Form form={infoForm} layout="vertical" initialValues={currentUser || {}} disabled={!editing}>
          <Row gutter={[20, 0]}>
            <Col xs={24} md={12}>
              <Form.Item name="username" label={intl.formatMessage({ id: 'profile.username' })} rules={[{ required: true, message: intl.formatMessage({ id: 'profile.usernameRule' }) }]}>
                <Input prefix={<UserOutlined className="profile-input-icon" />} placeholder={intl.formatMessage({ id: 'profile.username' })} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item
                name="email"
                label={intl.formatMessage({ id: 'profile.email' })}
                rules={[
                  { type: 'email', message: intl.formatMessage({ id: 'profile.emailInvalid' }) },
                  { required: true, message: intl.formatMessage({ id: 'profile.emailRequired' }) },
                ]}
              >
                <Input prefix={<MailOutlined className="profile-input-icon" />} placeholder={intl.formatMessage({ id: 'profile.email' })} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item name="realName" label={intl.formatMessage({ id: 'profile.realName' })}>
                <Input prefix={<IdcardOutlined className="profile-input-icon" />} placeholder={intl.formatMessage({ id: 'profile.realName' })} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item
                name="phone"
                label={intl.formatMessage({ id: 'profile.phoneLabel' })}
                rules={[{ pattern: /^1[3-9]\d{9}$/, message: intl.formatMessage({ id: 'profile.phoneInvalid' }) }]}
              >
                <Input placeholder={intl.formatMessage({ id: 'profile.phoneLabel' })} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item name="department" label={intl.formatMessage({ id: 'profile.dept' })}>
                <Input placeholder={intl.formatMessage({ id: 'profile.dept' })} />
              </Form.Item>
            </Col>
            <Col xs={24} md={12}>
              <Form.Item name="position" label={intl.formatMessage({ id: 'profile.position' })}>
                <Input placeholder={intl.formatMessage({ id: 'profile.position' })} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="description" label={intl.formatMessage({ id: 'profile.bio' })}>
            <Input.TextArea rows={4} placeholder={intl.formatMessage({ id: 'profile.bioPh' })} maxLength={500} showCount />
          </Form.Item>
        </Form>
      </Card>

      <Card className="profile-section-card profile-meta-card" title={intl.formatMessage({ id: 'profile.account' })}>
        <Row gutter={[16, 12]}>
          <Col xs={24} sm={8}>
            <Text type="secondary">{intl.formatMessage({ id: 'profile.userId' })}</Text>
            <div className="profile-meta-value">{currentUser?.id ?? '—'}</div>
          </Col>
          <Col xs={24} sm={8}>
            <Text type="secondary">{intl.formatMessage({ id: 'profile.registeredAt' })}</Text>
            <div className="profile-meta-value">{formatDateTime(currentUser?.createdAt || '') || '—'}</div>
          </Col>
          <Col xs={24} sm={8}>
            <Text type="secondary">{intl.formatMessage({ id: 'profile.lastLogin' })}</Text>
            <div className="profile-meta-value">
              {currentUser?.lastLoginAt ? formatDateTime(currentUser.lastLoginAt) : intl.formatMessage({ id: 'profile.neverLogin' })}
            </div>
          </Col>
        </Row>
      </Card>
    </div>
  );

  const passwordTabContent = (
    <div className="profile-tab-inner">
      <Card className="profile-section-card profile-password-card" title={intl.formatMessage({ id: 'profile.password' })}>
        <Text type="secondary" className="profile-password-hint">
          {intl.formatMessage({ id: 'profile.pwdHint' })}
        </Text>
        <Divider style={{ margin: '16px 0' }} />
        <Form form={pwdForm} layout="vertical" style={{ maxWidth: 440 }}>
          <Form.Item name="oldPassword" label={intl.formatMessage({ id: 'profile.oldPwd' })} rules={[{ required: true, message: intl.formatMessage({ id: 'profile.oldPwdRule' }) }]}>
            <Input.Password prefix={<LockOutlined />} placeholder={intl.formatMessage({ id: 'profile.oldPwd' })} />
          </Form.Item>
          <Form.Item
            name="newPassword"
            label={intl.formatMessage({ id: 'profile.newPwd' })}
            rules={[
              { required: true, message: intl.formatMessage({ id: 'profile.newPwdRule' }) },
              { min: 8, message: intl.formatMessage({ id: 'profile.newPwdMin' }) },
            ]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder={intl.formatMessage({ id: 'profile.newPwdPh' })} />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label={intl.formatMessage({ id: 'profile.confirmPwd' })}
            dependencies={['newPassword']}
            rules={[
              { required: true, message: intl.formatMessage({ id: 'profile.confirmPwdRule' }) },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error(intl.formatMessage({ id: 'profile.pwdMismatch' })));
                },
              }),
            ]}
          >
            <Input.Password prefix={<LockOutlined />} placeholder={intl.formatMessage({ id: 'profile.confirmPwdPh' })} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" icon={<SaveOutlined />} loading={pwdLoading} onClick={handlePasswordChange}>
              {intl.formatMessage({ id: 'profile.updatePwd' })}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );

  return (
    <PageContainer
      title={intl.formatMessage({ id: 'profile.pageTitle' })}
      subTitle={intl.formatMessage({ id: 'profile.pageSubtitle' })}
      className="profile-page-container"
    >
      <div className="profile-page">
        <Card className="profile-hero" bordered={false}>
          <div className="profile-hero__main">
            <Avatar size={72} className="profile-hero__avatar" icon={<UserOutlined />} />
            <div className="profile-hero__text">
              <div className="profile-hero__name">{currentUser?.username || intl.formatMessage({ id: 'app.user.fallback' })}</div>
              <Space size="middle" wrap className="profile-hero__meta">
                {currentUser?.email && (
                  <Text type="secondary">
                    <MailOutlined style={{ marginRight: 6 }} />
                    {currentUser.email}
                  </Text>
                )}
                <Text type="secondary">
                  {intl.formatMessage(
                    { id: 'profile.roleLabel' },
                    { role: currentUser?.role || intl.formatMessage({ id: 'profile.roleDefault' }) }
                  )}
                </Text>
              </Space>
            </div>
          </div>
        </Card>

        <Tabs
          className="profile-tabs"
          items={[
            { key: 'info', label: intl.formatMessage({ id: 'profile.tabInfo' }), children: infoTabContent },
            { key: 'password', label: intl.formatMessage({ id: 'profile.tabSecurity' }), children: passwordTabContent },
          ]}
        />
      </div>
    </PageContainer>
  );
};
