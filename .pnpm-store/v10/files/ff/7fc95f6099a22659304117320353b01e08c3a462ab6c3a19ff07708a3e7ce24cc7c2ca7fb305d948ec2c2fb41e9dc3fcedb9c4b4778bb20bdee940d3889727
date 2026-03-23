import { DivProps } from "../types/index.mjs";
import { CSSProperties } from "react";
import { NumberSize, Size } from "re-resizable";
import { Props } from "react-rnd";

//#region src/DraggablePanel/type.d.ts
type PlacementType = 'right' | 'left' | 'top' | 'bottom';
interface DraggablePanelProps extends DivProps {
  backgroundColor?: string;
  classNames?: {
    content?: string;
    handle?: string;
  };
  defaultExpand?: boolean;
  defaultSize?: Partial<Size>;
  destroyOnClose?: boolean;
  expand?: boolean;
  expandable?: boolean;
  fullscreen?: boolean;
  headerHeight?: number;
  maxHeight?: number;
  maxWidth?: number;
  minHeight?: number;
  minWidth?: number;
  mode?: 'fixed' | 'float';
  onExpandChange?: (expand: boolean) => void;
  onSizeChange?: (delta: NumberSize, size?: Size) => void;
  onSizeDragging?: (delta: NumberSize, size?: Size) => void;
  pin?: boolean;
  placement: PlacementType;
  resize?: Props['enableResizing'];
  /**
   * Whether to show border
   * @default true
   */
  showBorder?: boolean;
  showHandleHighlight?: boolean;
  showHandleWhenCollapsed?: boolean;
  showHandleWideArea?: boolean;
  size?: Partial<Size>;
  styles?: {
    content?: CSSProperties;
    handle?: CSSProperties;
  };
}
//#endregion
export { DraggablePanelProps };
//# sourceMappingURL=type.d.mts.map