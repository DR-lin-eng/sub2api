'use client';

import * as React from 'react';
import { useStore } from '@base-ui/utils/store';
import { useRenderElement } from "../../utils/useRenderElement.js";
import { useSelectRootContext } from "../root/SelectRootContext.js";
import { resolveSelectedLabel, resolveMultipleLabels } from "../../utils/resolveValueLabel.js";
import { selectors } from "../store.js";
const stateAttributesMapping = {
  value: () => null
};

/**
 * A text label of the currently selected item.
 * Renders a `<span>` element.
 *
 * Documentation: [Base UI Select](https://base-ui.com/react/components/select)
 */
export const SelectValue = /*#__PURE__*/React.forwardRef(function SelectValue(componentProps, forwardedRef) {
  const {
    className,
    render,
    children: childrenProp,
    ...elementProps
  } = componentProps;
  const {
    store,
    valueRef
  } = useSelectRootContext();
  const value = useStore(store, selectors.value);
  const items = useStore(store, selectors.items);
  const itemToStringLabel = useStore(store, selectors.itemToStringLabel);
  const serializedValue = useStore(store, selectors.serializedValue);
  const state = React.useMemo(() => ({
    value,
    placeholder: !serializedValue
  }), [value, serializedValue]);
  const children = typeof childrenProp === 'function' ? childrenProp(value) : childrenProp ?? (Array.isArray(value) ? resolveMultipleLabels(value, itemToStringLabel) : resolveSelectedLabel(value, items, itemToStringLabel));
  const element = useRenderElement('span', componentProps, {
    state,
    ref: [forwardedRef, valueRef],
    props: [{
      children
    }, elementProps],
    stateAttributesMapping
  });
  return element;
});
if (process.env.NODE_ENV !== "production") SelectValue.displayName = "SelectValue";