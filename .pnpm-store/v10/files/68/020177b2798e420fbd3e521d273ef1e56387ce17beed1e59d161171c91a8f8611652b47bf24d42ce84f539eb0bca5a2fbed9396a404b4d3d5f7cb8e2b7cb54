'use client';

import * as React from 'react';
import * as ReactDOM from 'react-dom';
import { useControlled } from '@base-ui/utils/useControlled';
import { useStableCallback } from '@base-ui/utils/useStableCallback';
import { safePolygon, useDismiss, useHover, useInteractions, useFloatingRootContext } from "../../floating-ui-react/index.js";
import { PreviewCardRootContext } from "./PreviewCardContext.js";
import { CLOSE_DELAY, OPEN_DELAY } from "../utils/constants.js";
import { REASONS } from "../../utils/reasons.js";
import { useFocusWithDelay } from "../../utils/interactions/useFocusWithDelay.js";
import { useOpenChangeComplete } from "../../utils/useOpenChangeComplete.js";
import { useTransitionStatus } from "../../utils/useTransitionStatus.js";

/**
 * Groups all parts of the preview card.
 * Doesn’t render its own HTML element.
 *
 * Documentation: [Base UI Preview Card](https://base-ui.com/react/components/preview-card)
 */
import { jsx as _jsx } from "react/jsx-runtime";
export function PreviewCardRoot(props) {
  const {
    open: externalOpen,
    defaultOpen,
    onOpenChange: onOpenChangeProp,
    onOpenChangeComplete,
    actionsRef
  } = props;
  const delayRef = React.useRef(OPEN_DELAY);
  const closeDelayRef = React.useRef(CLOSE_DELAY);
  const writeDelayRefs = useStableCallback(config => {
    delayRef.current = config.delay ?? OPEN_DELAY;
    closeDelayRef.current = config.closeDelay ?? CLOSE_DELAY;
  });
  const [triggerElement, setTriggerElement] = React.useState(null);
  const [positionerElement, setPositionerElement] = React.useState(null);
  const [instantTypeState, setInstantTypeState] = React.useState();
  const popupRef = React.useRef(null);
  const [open, setOpenUnwrapped] = useControlled({
    controlled: externalOpen,
    default: defaultOpen,
    name: 'PreviewCard',
    state: 'open'
  });
  const onOpenChange = useStableCallback(onOpenChangeProp);
  const {
    mounted,
    setMounted,
    transitionStatus
  } = useTransitionStatus(open);
  const handleUnmount = useStableCallback(() => {
    setMounted(false);
    onOpenChangeComplete?.(false);
  });
  useOpenChangeComplete({
    enabled: !actionsRef,
    open,
    ref: popupRef,
    onComplete() {
      if (!open) {
        handleUnmount();
      }
    }
  });
  React.useImperativeHandle(actionsRef, () => ({
    unmount: handleUnmount
  }), [handleUnmount]);
  const setOpen = useStableCallback((nextOpen, eventDetails) => {
    const isHover = eventDetails.reason === REASONS.triggerHover;
    const isFocusOpen = nextOpen && eventDetails.reason === REASONS.triggerFocus;
    const isDismissClose = !nextOpen && (eventDetails.reason === REASONS.triggerPress || eventDetails.reason === REASONS.escapeKey);
    onOpenChange(nextOpen, eventDetails);
    if (eventDetails.isCanceled) {
      return;
    }
    function changeState() {
      setOpenUnwrapped(nextOpen);
    }
    if (isHover) {
      // If a hover reason is provided, we need to flush the state synchronously. This ensures
      // `node.getAnimations()` knows about the new state.
      ReactDOM.flushSync(changeState);
    } else {
      changeState();
    }
    if (isFocusOpen || isDismissClose) {
      setInstantTypeState(isFocusOpen ? 'focus' : 'dismiss');
    } else if (eventDetails.reason === REASONS.triggerHover) {
      setInstantTypeState(undefined);
    }
  });
  const context = useFloatingRootContext({
    elements: {
      reference: triggerElement,
      floating: positionerElement
    },
    open,
    onOpenChange: setOpen
  });
  const instantType = instantTypeState;
  const getDelayValue = () => delayRef.current;
  const getCloseDelayValue = () => closeDelayRef.current;
  const hover = useHover(context, {
    mouseOnly: true,
    move: false,
    handleClose: safePolygon(),
    restMs: getDelayValue,
    delay: () => ({
      close: getCloseDelayValue()
    })
  });
  const focus = useFocusWithDelay(context, {
    delay: getDelayValue
  });
  const dismiss = useDismiss(context);
  const {
    getReferenceProps,
    getFloatingProps
  } = useInteractions([hover, focus, dismiss]);
  const contextValue = React.useMemo(() => ({
    open,
    setOpen,
    mounted,
    setMounted,
    setTriggerElement,
    positionerElement,
    setPositionerElement,
    popupRef,
    triggerProps: getReferenceProps(),
    popupProps: getFloatingProps(),
    floatingRootContext: context,
    instantType,
    transitionStatus,
    onOpenChangeComplete,
    writeDelayRefs
  }), [open, setOpen, mounted, setMounted, positionerElement, getReferenceProps, getFloatingProps, context, instantType, transitionStatus, onOpenChangeComplete, writeDelayRefs]);
  return /*#__PURE__*/_jsx(PreviewCardRootContext.Provider, {
    value: contextValue,
    children: props.children
  });
}