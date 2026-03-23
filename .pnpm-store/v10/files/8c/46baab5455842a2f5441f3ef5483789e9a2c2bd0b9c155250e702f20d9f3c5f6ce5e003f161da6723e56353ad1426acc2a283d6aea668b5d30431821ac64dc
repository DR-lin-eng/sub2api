import * as React from 'react';
type ItemRecord = Record<string, React.ReactNode>;
type ItemsInput = ItemRecord | ReadonlyArray<LabeledItem> | ReadonlyArray<Group<LabeledItem>> | undefined;
interface LabeledItem {
  value: any;
  label: React.ReactNode;
}
export interface Group<Item = any> {
  value: unknown;
  items: Item[];
}
export declare function isGroupedItems(items: ReadonlyArray<any | Group<any>> | undefined): items is Group<any>[];
export declare function stringifyAsLabel(item: any, itemToStringLabel?: (item: any) => string): string;
export declare function stringifyAsValue(item: any, itemToStringValue?: (item: any) => string): string;
export declare function resolveSelectedLabel(value: any, items: ItemsInput, itemToStringLabel?: (item: any) => string): React.ReactNode;
export declare function resolveMultipleLabels(values: any[] | undefined, itemToStringLabel?: (item: any) => string): string;
export {};