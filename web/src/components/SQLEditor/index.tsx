import Editor from '@monaco-editor/react';
import { useState } from 'react';
import { useIntl } from '@/locales';

interface SQLEditorProps {
  value?: string;
  onChange?: (value: string) => void;
  height?: string;
  readOnly?: boolean;
  placeholder?: string;
}

export default function SQLEditor({
  value = '',
  onChange,
  height = '300px',
  readOnly = false,
  placeholder,
}: SQLEditorProps) {
  const intl = useIntl();
  const ph = placeholder ?? intl.formatMessage({ id: 'queryEngine.rule.sql' });
  const [editorValue, setEditorValue] = useState(value);

  const handleChange = (newValue: string | undefined) => {
    const val = newValue || '';
    setEditorValue(val);
    onChange?.(val);
  };

  return (
    <div title={ph} style={{ border: '1px solid #d9d9d9', borderRadius: '4px', overflow: 'hidden' }}>
      <Editor
        height={height}
        defaultLanguage="sql"
        value={editorValue || value}
        onChange={handleChange}
        theme="vs"
        options={{
          readOnly,
          minimap: { enabled: false },
          fontSize: 14,
          lineNumbers: 'on',
          scrollBeyondLastLine: false,
          wordWrap: 'on',
          automaticLayout: true,
        }}
      />
    </div>
  );
}
