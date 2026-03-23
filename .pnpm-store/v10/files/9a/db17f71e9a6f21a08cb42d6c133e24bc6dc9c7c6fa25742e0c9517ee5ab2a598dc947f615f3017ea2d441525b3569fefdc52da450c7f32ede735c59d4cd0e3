'use client';

import * as React from 'react';
import { useIsoLayoutEffect } from '@base-ui/utils/useIsoLayoutEffect';
import { useScrollAreaViewportContext } from "../viewport/ScrollAreaViewportContext.js";
import { useRenderElement } from "../../utils/useRenderElement.js";
import { useScrollAreaRootContext } from "../root/ScrollAreaRootContext.js";
import { scrollAreaStateAttributesMapping } from "../root/stateAttributes.js";
/**
 * A container for the content of the scroll area.
 * Renders a `<div>` element.
 *
 * Documentation: [Base UI Scroll Area](https://base-ui.com/react/components/scroll-area)
 */
export const ScrollAreaContent = /*#__PURE__*/React.forwardRef(function ScrollAreaContent(componentProps, forwardedRef) {
  const {
    render,
    className,
    ...elementProps
  } = componentProps;
  const contentWrapperRef = React.useRef(null);
  const {
    computeThumbPosition
  } = useScrollAreaViewportContext();
  const {
    viewportState
  } = useScrollAreaRootContext();
  useIsoLayoutEffect(() => {
    if (typeof ResizeObserver === 'undefined') {
      return undefined;
    }
    const ro = new ResizeObserver(computeThumbPosition);
    if (contentWrapperRef.current) {
      ro.observe(contentWrapperRef.current);
    }
    return () => {
      ro.disconnect();
    };
  }, [computeThumbPosition]);
  const element = useRenderElement('div', componentProps, {
    ref: [forwardedRef, contentWrapperRef],
    state: viewportState,
    stateAttributesMapping: scrollAreaStateAttributesMapping,
    props: [{
      role: 'presentation',
      style: {
        minWidth: 'fit-content'
      }
    }, elementProps]
  });
  return element;
});
if (process.env.NODE_ENV !== "production") ScrollAreaContent.displayName = "ScrollAreaContent";