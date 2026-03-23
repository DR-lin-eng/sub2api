'use client';

import * as React from 'react';
import { useStore } from '@base-ui/utils/store';
import { useComboboxRootContext } from "../root/ComboboxRootContext.js";
import { resolveSelectedLabel } from "../../utils/resolveValueLabel.js";
import { selectors } from "../store.js";

/**
 * The current value of the combobox.
 * Doesn't render its own HTML element.
 *
 * Documentation: [Base UI Combobox](https://base-ui.com/react/components/combobox)
 */
import { jsx as _jsx } from "react/jsx-runtime";
export function ComboboxValue(props) {
  const {
    children: childrenProp
  } = props;
  const store = useComboboxRootContext();
  const itemToStringLabel = useStore(store, selectors.itemToStringLabel);
  const selectedValue = useStore(store, selectors.selectedValue);
  const items = useStore(store, selectors.items);
  let returnValue = null;
  if (typeof childrenProp === 'function') {
    returnValue = childrenProp(selectedValue);
  } else if (childrenProp != null) {
    returnValue = childrenProp;
  } else {
    returnValue = resolveSelectedLabel(selectedValue, items, itemToStringLabel);
  }
  return /*#__PURE__*/_jsx(React.Fragment, {
    children: returnValue
  });
}