import { useCallback, useEffect, useRef, useState } from 'react';
import Editor from '@monaco-editor/react';
import type { editor } from 'monaco-editor';

type Props = {
  value: string;
  /** 至少高度（px） */
  minHeight?: number;
  /** 最大高度（px），超出后由编辑器内滚动条承接 */
  maxHeight?: number;
};

/**
 * 只读 SQL 展示：高度随内容变化，避免短 SQL 大块留白或长 SQL 被裁切。
 */
export default function AutoHeightSqlEditor({
  value,
  minHeight = 96,
  maxHeight = 640,
}: Props) {
  const editorRef = useRef<editor.IStandaloneCodeEditor | null>(null);
  const [height, setHeight] = useState(minHeight);

  const fitHeight = useCallback(() => {
    const ed = editorRef.current;
    if (!ed) return;
    requestAnimationFrame(() => {
      ed.layout();
      const contentH = ed.getContentHeight();
      const next = Math.min(maxHeight, Math.max(minHeight, contentH));
      setHeight(next);
    });
  }, [minHeight, maxHeight]);

  useEffect(() => {
    fitHeight();
  }, [value, fitHeight]);

  return (
    <Editor
      height={height}
      language="sql"
      theme="vs"
      value={value ?? ''}
      options={{
        readOnly: true,
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        wordWrap: 'on',
        fontSize: 13,
        lineNumbers: 'on',
        scrollbar: {
          vertical: 'auto',
          horizontal: 'auto',
        },
        overviewRulerLanes: 0,
        hideCursorInOverviewRuler: true,
        overviewRulerBorder: false,
      }}
      onMount={(ed) => {
        editorRef.current = ed;
        fitHeight();
        ed.onDidContentSizeChange(() => fitHeight());
      }}
    />
  );
}
