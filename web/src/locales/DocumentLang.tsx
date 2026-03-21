import { useEffect } from 'react';
import { useIntl } from 'umi';

/** 同步 <html lang>，便于无障碍与字体回退。 */
export default function DocumentLang() {
  const intl = useIntl();
  useEffect(() => {
    document.documentElement.lang = intl.locale.startsWith('zh') ? 'zh-CN' : 'en';
  }, [intl.locale]);
  return null;
}
