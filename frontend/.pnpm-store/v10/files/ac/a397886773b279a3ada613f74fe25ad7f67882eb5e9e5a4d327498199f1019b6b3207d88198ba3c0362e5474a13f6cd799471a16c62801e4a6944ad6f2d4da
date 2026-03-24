'use client';

import * as React from 'react';
import { useTooltipRootContext } from "../root/TooltipRootContext.js";
import { useTooltipPositionerContext } from "../positioner/TooltipPositionerContext.js";
import { popupStateMapping as baseMapping } from "../../utils/popupStateMapping.js";
import { transitionStatusMapping } from "../../utils/stateAttributesMapping.js";
import { useOpenChangeComplete } from "../../utils/useOpenChangeComplete.js";
import { useRenderElement } from "../../utils/useRenderElement.js";
import { usePopupAutoResize } from "../../utils/usePopupAutoResize.js";
import { getDisabledMountTransitionStyles } from "../../utils/getDisabledMountTransitionStyles.js";
import { useHoverFloatingInteraction } from "../../floating-ui-react/index.js";
import { useDirection } from "../../direction-provider/index.js";
const stateAttributesMapping = {
  ...baseMapping,
  ...transitionStatusMapping
};

/**
 * A container for the tooltip contents.
 * Renders a `<div>` element.
 *
 * Documentation: [Base UI Tooltip](https://base-ui.com/react/components/tooltip)
 */
export const TooltipPopup = /*#__PURE__*/React.forwardRef(function TooltipPopup(componentProps, forwardedRef) {
  const {
    className,
    render,
    ...elementProps
  } = componentProps;
  const store = useTooltipRootContext();
  const {
    side,
    align
  } = useTooltipPositionerContext();
  const open = store.useState('open');
  const mounted = store.useState('mounted');
  const instantType = store.useState('instantType');
  const transitionStatus = store.useState('transitionStatus');
  const popupProps = store.useState('popupProps');
  const payload = store.useState('payload');
  const popupElement = store.useState('popupElement');
  const positionerElement = store.useState('positionerElement');
  const floatingContext = store.useState('floatingRootContext');
  const direction = useDirection();
  useOpenChangeComplete({
    open,
    ref: store.context.popupRef,
    onComplete() {
      if (open) {
        store.context.onOpenChangeComplete?.(true);
      }
    }
  });
  function handleMeasureLayout() {
    floatingContext.context.events.emit('measure-layout');
  }
  function handleMeasureLayoutComplete(previousDimensions, nextDimensions) {
    floatingContext.context.events.emit('measure-layout-complete', {
      previousDimensions,
      nextDimensions
    });
  }

  // If there's just one trigger, we can skip the auto-resize logic as
  // the tooltip will always be anchored to the same position.
  const autoresizeEnabled = React.useCallback(() => store.context.triggerElements.size > 1, [store]);
  usePopupAutoResize({
    popupElement,
    positionerElement,
    mounted,
    content: payload,
    enabled: autoresizeEnabled,
    onMeasureLayout: handleMeasureLayout,
    onMeasureLayoutComplete: handleMeasureLayoutComplete,
    side,
    direction
  });
  const disabled = store.useState('disabled');
  const closeDelay = store.useState('closeDelay');
  useHoverFloatingInteraction(floatingContext, {
    enabled: !disabled,
    closeDelay
  });
  const state = React.useMemo(() => ({
    open,
    side,
    align,
    instant: instantType,
    transitionStatus
  }), [open, side, align, instantType, transitionStatus]);
  const element = useRenderElement('div', componentProps, {
    state,
    ref: [forwardedRef, store.context.popupRef, store.useStateSetter('popupElement')],
    props: [popupProps, getDisabledMountTransitionStyles(transitionStatus), elementProps],
    stateAttributesMapping
  });
  return element;
});
if (process.env.NODE_ENV !== "production") TooltipPopup.displayName = "TooltipPopup";