"use strict";
'use client';

var _interopRequireWildcard = require("@babel/runtime/helpers/interopRequireWildcard").default;
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.PreviewCardTrigger = void 0;
var React = _interopRequireWildcard(require("react"));
var _useIsoLayoutEffect = require("@base-ui/utils/useIsoLayoutEffect");
var _PreviewCardContext = require("../root/PreviewCardContext");
var _popupStateMapping = require("../../utils/popupStateMapping");
var _useRenderElement = require("../../utils/useRenderElement");
/**
 * A link that opens the preview card.
 * Renders an `<a>` element.
 *
 * Documentation: [Base UI Preview Card](https://base-ui.com/react/components/preview-card)
 */
const PreviewCardTrigger = exports.PreviewCardTrigger = /*#__PURE__*/React.forwardRef(function PreviewCardTrigger(componentProps, forwardedRef) {
  const {
    render,
    className,
    delay,
    closeDelay,
    ...elementProps
  } = componentProps;
  const {
    open,
    triggerProps,
    setTriggerElement,
    writeDelayRefs
  } = (0, _PreviewCardContext.usePreviewCardRootContext)();
  (0, _useIsoLayoutEffect.useIsoLayoutEffect)(() => {
    writeDelayRefs({
      delay,
      closeDelay
    });
  });
  const state = React.useMemo(() => ({
    open
  }), [open]);
  const element = (0, _useRenderElement.useRenderElement)('a', componentProps, {
    ref: [setTriggerElement, forwardedRef],
    state,
    props: [triggerProps, elementProps],
    stateAttributesMapping: _popupStateMapping.triggerOpenStateMapping
  });
  return element;
});
if (process.env.NODE_ENV !== "production") PreviewCardTrigger.displayName = "PreviewCardTrigger";