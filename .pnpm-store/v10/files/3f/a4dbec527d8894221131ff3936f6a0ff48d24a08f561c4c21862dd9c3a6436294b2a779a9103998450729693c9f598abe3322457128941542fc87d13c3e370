'use client';

import { serializeValue } from "./serializeValue.js";
export function isGroupedItems(items) {
  return items != null && items.length > 0 && typeof items[0] === 'object' && items[0] != null && 'items' in items[0];
}
export function stringifyAsLabel(item, itemToStringLabel) {
  if (itemToStringLabel && item != null) {
    return itemToStringLabel(item) ?? '';
  }
  if (item && typeof item === 'object') {
    if ('label' in item && item.label != null) {
      return String(item.label);
    }
    if ('value' in item) {
      return String(item.value);
    }
  }
  return serializeValue(item);
}
export function stringifyAsValue(item, itemToStringValue) {
  if (itemToStringValue && item != null) {
    return itemToStringValue(item) ?? '';
  }
  if (item && typeof item === 'object' && 'value' in item && 'label' in item) {
    return serializeValue(item.value);
  }
  return serializeValue(item);
}
export function resolveSelectedLabel(value, items, itemToStringLabel) {
  if (itemToStringLabel && value != null) {
    return itemToStringLabel(value);
  }

  // Custom object with explicit label takes precedence
  if (value && typeof value === 'object' && 'label' in value && value.label != null) {
    return value.label;
  }

  // Items provided as plain record map
  if (items && !Array.isArray(items)) {
    return items[value] ?? stringifyAsLabel(value, itemToStringLabel);
  }

  // Items provided as array (flat or grouped)
  if (Array.isArray(items)) {
    const flatItems = isGroupedItems(items) ? items.flatMap(g => g.items) : items;

    // If no value selected, prefer the null option label when available
    if (value == null) {
      const nullItem = flatItems.find(it => it.value == null);
      if (nullItem && nullItem.label != null) {
        return nullItem.label;
      }
      return stringifyAsLabel(value, itemToStringLabel);
    }

    // Primitive selected value: map to first matching item's label
    if (typeof value !== 'object') {
      const match = flatItems.find(it => it && it.value === value);
      if (match && match.label != null) {
        return match.label;
      }
      return stringifyAsLabel(value, itemToStringLabel);
    }

    // Object without explicit label: try matching by its `value` property
    if ('value' in value) {
      const match = flatItems.find(it => it && it.value === value.value);
      if (match && match.label != null) {
        return match.label;
      }
    }
  }
  return stringifyAsLabel(value, itemToStringLabel);
}
export function resolveMultipleLabels(values, itemToStringLabel) {
  if (!Array.isArray(values) || values.length === 0) {
    return '';
  }
  return values.map(v => stringifyAsLabel(v, itemToStringLabel)).join(', ');
}