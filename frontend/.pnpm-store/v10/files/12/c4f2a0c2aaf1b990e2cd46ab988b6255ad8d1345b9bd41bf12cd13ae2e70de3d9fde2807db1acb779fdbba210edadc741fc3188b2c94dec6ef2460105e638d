import { ContextMenuItem } from "./type.mjs";
import React, { HTMLAttributes, MouseEvent, ReactNode } from "react";

//#region src/ContextMenu/ContextMenuTrigger.d.ts
type ContextMenuTriggerProps = {
  children: ReactNode;
  /**
   * Menu items to display. Supports lazy rendering via function.
   * When provided, context menu will be automatically shown on right-click.
   */
  items?: ContextMenuItem[] | (() => ContextMenuItem[]);
  /**
   * Custom context menu handler. If `items` is provided, this is optional.
   */
  onContextMenu?: (event: MouseEvent<HTMLElement>) => void;
} & Omit<HTMLAttributes<HTMLElement>, 'onContextMenu' | 'children'>;
declare const ContextMenuTrigger: React.NamedExoticComponent<ContextMenuTriggerProps>;
//#endregion
export { ContextMenuTrigger, ContextMenuTriggerProps };
//# sourceMappingURL=ContextMenuTrigger.d.mts.map