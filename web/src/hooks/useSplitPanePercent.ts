import { useCallback, useRef, useState } from 'react';

type Options = {
  /** 初始宽度占父容器百分比 */
  defaultPct: number;
  minPct: number;
  maxPct: number;
  storageKey?: string;
};

/**
 * 左侧分栏占父容器宽度百分比，通过分隔条拖拽调整；可选写入 localStorage。
 */
export function useSplitPanePercent({ defaultPct, minPct, maxPct, storageKey }: Options) {
  const [percent, setPercent] = useState(() => {
    if (!storageKey) return defaultPct;
    try {
      const n = Number(localStorage.getItem(storageKey));
      if (Number.isFinite(n) && n >= minPct && n <= maxPct) return n;
    } catch {
      /* ignore */
    }
    return defaultPct;
  });
  const [dragging, setDragging] = useState(false);
  const percentRef = useRef(percent);
  percentRef.current = percent;

  const persist = useCallback(
    (v: number) => {
      if (!storageKey) return;
      try {
        localStorage.setItem(storageKey, String(v));
      } catch {
        /* ignore */
      }
    },
    [storageKey],
  );

  const onSplitterMouseDown = useCallback(
    (e: React.MouseEvent<HTMLDivElement>) => {
      e.preventDefault();
      const container = e.currentTarget.parentElement;
      if (!container) return;
      setDragging(true);
      const startX = e.clientX;
      const startPct = percentRef.current;

      const onMove = (ev: MouseEvent) => {
        const w = container.getBoundingClientRect().width;
        if (w <= 0) return;
        const delta = ((ev.clientX - startX) / w) * 100;
        const next = Math.min(maxPct, Math.max(minPct, startPct + delta));
        percentRef.current = next;
        setPercent(next);
      };

      const onUp = () => {
        setDragging(false);
        document.removeEventListener('mousemove', onMove);
        document.removeEventListener('mouseup', onUp);
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
        persist(percentRef.current);
      };

      document.addEventListener('mousemove', onMove);
      document.addEventListener('mouseup', onUp);
      document.body.style.cursor = 'col-resize';
      document.body.style.userSelect = 'none';
    },
    [minPct, maxPct, persist],
  );

  return { percent, dragging, onSplitterMouseDown };
}
