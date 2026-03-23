'use client';

import * as React from 'react';
import { usePreviewCardRootContext } from "../root/PreviewCardContext.js";
import { usePreviewCardPositionerContext } from "../positioner/PreviewCardPositionerContext.js";
import { popupStateMapping as baseMapping } from "../../utils/popupStateMapping.js";
import { transitionStatusMapping } from "../../utils/stateAttributesMapping.js";
import { useOpenChangeComplete } from "../../utils/useOpenChangeComplete.js";
import { useRenderElement } from "../../utils/useRenderElement.js";
import { getDisabledMountTransitionStyles } from "../../utils/getDisabledMountTransitionStyles.js";
const stateAttributesMapping = {
  ...baseMapping,
  ...transitionStatusMapping
};

/**
 * A container for the preview card contents.
 * Renders a `<div>` element.
 *
 * Documentation: [Base UI Preview Card](https://base-ui.com/react/components/preview-card)
 */
export const PreviewCardPopup = /*#__PURE__*/React.forwardRef(function PreviewCardPopup(componentProps, forwardedRef) {
  const {
    className,
    render,
    ...elementProps
  } = componentProps;
  const {
    open,
    transitionStatus,
    popupRef,
    onOpenChangeComplete,
    popupProps
  } = usePreviewCardRootContext();
  const {
    side,
    align
  } = usePreviewCardPositionerContext();
  useOpenChangeComplete({
    open,
    ref: popupRef,
    onComplete() {
      if (open) {
        onOpenChangeComplete?.(true);
      }
    }
  });
  const state = React.useMemo(() => ({
    open,
    side,
    align,
    transitionStatus
  }), [open, side, align, transitionStatus]);
  const element = useRenderElement('div', componentProps, {
    ref: [popupRef, forwardedRef],
    state,
    props: [popupProps, getDisabledMountTransitionStyles(transitionStatus), elementProps],
    stateAttributesMapping
  });
  return element;
});
if (process.env.NODE_ENV !== "production") PreviewCardPopup.displayName = "PreviewCardPopup";