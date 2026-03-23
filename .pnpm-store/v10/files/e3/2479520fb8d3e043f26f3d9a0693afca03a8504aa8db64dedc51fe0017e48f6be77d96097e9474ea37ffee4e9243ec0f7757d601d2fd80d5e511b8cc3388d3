"use strict";
'use client';

var _interopRequireWildcard = require("@babel/runtime/helpers/interopRequireWildcard").default;
Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.useOpenInteractionType = useOpenInteractionType;
var React = _interopRequireWildcard(require("react"));
var _useStableCallback = require("@base-ui/utils/useStableCallback");
var _useEnhancedClickHandler = require("@base-ui/utils/useEnhancedClickHandler");
/**
 * Determines the interaction type (keyboard, mouse, touch, etc.) that opened the component.
 *
 * @param open The open state of the component.
 */
function useOpenInteractionType(open) {
  const [openMethod, setOpenMethod] = React.useState(null);
  const handleTriggerClick = (0, _useStableCallback.useStableCallback)((_, interactionType) => {
    if (!open) {
      setOpenMethod(interactionType);
    }
  });
  const reset = React.useCallback(() => {
    setOpenMethod(null);
  }, []);
  const {
    onClick,
    onPointerDown
  } = (0, _useEnhancedClickHandler.useEnhancedClickHandler)(handleTriggerClick);
  return React.useMemo(() => ({
    openMethod,
    reset,
    triggerProps: {
      onClick,
      onPointerDown
    }
  }), [openMethod, reset, onClick, onPointerDown]);
}