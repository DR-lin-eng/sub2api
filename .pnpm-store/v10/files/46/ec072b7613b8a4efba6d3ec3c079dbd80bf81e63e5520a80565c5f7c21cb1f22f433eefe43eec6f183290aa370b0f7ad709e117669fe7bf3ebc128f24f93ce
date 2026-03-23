"use strict";
'use client';

var _interopRequireWildcard = require("@babel/runtime/helpers/interopRequireWildcard").default;
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.SelectValue = void 0;
var React = _interopRequireWildcard(require("react"));
var _store = require("@base-ui/utils/store");
var _useRenderElement = require("../../utils/useRenderElement");
var _SelectRootContext = require("../root/SelectRootContext");
var _resolveValueLabel = require("../../utils/resolveValueLabel");
var _store2 = require("../store");
const stateAttributesMapping = {
  value: () => null
};

/**
 * A text label of the currently selected item.
 * Renders a `<span>` element.
 *
 * Documentation: [Base UI Select](https://base-ui.com/react/components/select)
 */
const SelectValue = exports.SelectValue = /*#__PURE__*/React.forwardRef(function SelectValue(componentProps, forwardedRef) {
  const {
    className,
    render,
    children: childrenProp,
    ...elementProps
  } = componentProps;
  const {
    store,
    valueRef
  } = (0, _SelectRootContext.useSelectRootContext)();
  const value = (0, _store.useStore)(store, _store2.selectors.value);
  const items = (0, _store.useStore)(store, _store2.selectors.items);
  const itemToStringLabel = (0, _store.useStore)(store, _store2.selectors.itemToStringLabel);
  const serializedValue = (0, _store.useStore)(store, _store2.selectors.serializedValue);
  const state = React.useMemo(() => ({
    value,
    placeholder: !serializedValue
  }), [value, serializedValue]);
  const children = typeof childrenProp === 'function' ? childrenProp(value) : childrenProp ?? (Array.isArray(value) ? (0, _resolveValueLabel.resolveMultipleLabels)(value, itemToStringLabel) : (0, _resolveValueLabel.resolveSelectedLabel)(value, items, itemToStringLabel));
  const element = (0, _useRenderElement.useRenderElement)('span', componentProps, {
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