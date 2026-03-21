import type { FC } from 'react';
import { Select } from 'antd';
import { GlobalOutlined } from '@ant-design/icons';
import { useIntl, setLocale } from '@/locales';

const LanguageSwitch: FC = () => {
  const intl = useIntl();
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      <GlobalOutlined style={{ color: '#8c8c8c' }} />
      <Select
        size="small"
        value={intl.locale}
        style={{ width: 120 }}
        options={[
          { value: 'zh-CN', label: intl.formatMessage({ id: 'lang.zh' }) },
          { value: 'en-US', label: intl.formatMessage({ id: 'lang.en' }) },
        ]}
        onChange={(v) => setLocale(v as string, false)}
      />
    </div>
  );
};

export default LanguageSwitch;
