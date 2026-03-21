import { Result, Button } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useIntl } from '@/locales';
import DocumentLang from '@/locales/DocumentLang';

export default () => {
  const navigate = useNavigate();
  const intl = useIntl();

  return (
    <>
      <DocumentLang />
      <Result
        status="404"
        title="404"
        subTitle={intl.formatMessage({ id: 'notFound.subtitle' })}
        extra={
          <Button type="primary" onClick={() => navigate('/home')}>
            {intl.formatMessage({ id: 'notFound.backHome' })}
          </Button>
        }
      />
    </>
  );
};
