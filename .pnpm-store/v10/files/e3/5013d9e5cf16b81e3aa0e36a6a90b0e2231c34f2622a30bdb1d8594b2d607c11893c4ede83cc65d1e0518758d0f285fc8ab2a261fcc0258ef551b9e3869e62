import { FlexboxProps } from "../Flex/type.mjs";
import "../Flex/index.mjs";
import { ReactNode, Ref } from "react";

//#region src/SortableList/type.d.ts
interface SortableListItem {
  [key: string]: any;
  id: string | number;
}
interface SortableListProps extends Omit<FlexboxProps, 'onChange'> {
  items: SortableListItem[];
  onChange(items: SortableListItem[]): void;
  ref?: Ref<HTMLUListElement>;
  renderItem(item: SortableListItem): ReactNode;
}
//#endregion
export { SortableListItem, SortableListProps };
//# sourceMappingURL=type.d.mts.map