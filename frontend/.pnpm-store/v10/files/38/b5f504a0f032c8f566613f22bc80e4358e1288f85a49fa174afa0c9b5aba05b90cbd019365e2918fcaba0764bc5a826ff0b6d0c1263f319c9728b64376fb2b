"use strict";
'use client';

var _interopRequireWildcard = require("@babel/runtime/helpers/interopRequireWildcard").default;
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.ComboboxValue = ComboboxValue;
var React = _interopRequireWildcard(require("react"));
var _store = require("@base-ui/utils/store");
var _ComboboxRootContext = require("../root/ComboboxRootContext");
var _resolveValueLabel = require("../../utils/resolveValueLabel");
var _store2 = require("../store");
var _jsxRuntime = require("react/jsx-runtime");
/**
 * The current value of the combobox.
 * Doesn't render its own HTML element.
 *
 * Documentation: [Base UI Combobox](https://base-ui.com/react/components/combobox)
 */
function ComboboxValue(props) {
  const {
    children: childrenProp
  } = props;
  const store = (0, _ComboboxRootContext.useComboboxRootContext)();
  const itemToStringLabel = (0, _store.useStore)(store, _store2.selectors.itemToStringLabel);
  const selectedValue = (0, _store.useStore)(store, _store2.selectors.selectedValue);
  const items = (0, _store.useStore)(store, _store2.selectors.items);
  let returnValue = null;
  if (typeof childrenProp === 'function') {
    returnValue = childrenProp(selectedValue);
  } else if (childrenProp != null) {
    returnValue = childrenProp;
  } else {
    returnValue = (0, _resolveValueLabel.resolveSelectedLabel)(selectedValue, items, itemToStringLabel);
  }
  return /*#__PURE__*/(0, _jsxRuntime.jsx)(React.Fragment, {
    children: returnValue
  });
}