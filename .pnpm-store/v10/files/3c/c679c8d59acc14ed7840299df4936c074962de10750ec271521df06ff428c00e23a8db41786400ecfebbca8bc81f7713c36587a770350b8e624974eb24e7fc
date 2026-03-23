'use client';

import * as React from 'react';
import { useIsoLayoutEffect } from '@base-ui/utils/useIsoLayoutEffect';
import { usePreviewCardRootContext } from "../root/PreviewCardContext.js";
import { triggerOpenStateMapping } from "../../utils/popupStateMapping.js";
import { useRenderElement } from "../../utils/useRenderElement.js";

/**
 * A link that opens the preview card.
 * Renders an `<a>` element.
 *
 * Documentation: [Base UI Preview Card](https://base-ui.com/react/components/preview-card)
 */
export const PreviewCardTrigger = /*#__PURE__*/React.forwardRef(function PreviewCardTrigger(componentProps, forwardedRef) {
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
  } = usePreviewCardRootContext();
  useIsoLayoutEffect(() => {
    writeDelayRefs({
      delay,
      closeDelay
    });
  });
  const state = React.useMemo(() => ({
    open
  }), [open]);
  const element = useRenderElement('a', componentProps, {
    ref: [setTriggerElement, forwardedRef],
    state,
    props: [triggerProps, elementProps],
    stateAttributesMapping: triggerOpenStateMapping
  });
  return element;
});
if (process.env.NODE_ENV !== "production") PreviewCardTrigger.displayName = "PreviewCardTrigger";