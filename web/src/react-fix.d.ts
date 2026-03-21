// Fix React 18 type compatibility with @ant-design/pro-components
// ReactPortal in @types/react@18 has stricter types incompatible with ProComponents
import 'react';

declare module 'react' {
  interface ReactPortal {
    children?: ReactNode;
    key: Key | null;
  }
}
