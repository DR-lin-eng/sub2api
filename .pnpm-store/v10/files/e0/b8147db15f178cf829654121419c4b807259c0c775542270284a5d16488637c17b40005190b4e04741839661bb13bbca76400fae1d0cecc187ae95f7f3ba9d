'use client';

import * as React from 'react';
import { useStableCallback } from '@base-ui/utils/useStableCallback';
import { useEnhancedClickHandler } from '@base-ui/utils/useEnhancedClickHandler';

/**
 * Determines the interaction type (keyboard, mouse, touch, etc.) that opened the component.
 *
 * @param open The open state of the component.
 */
export function useOpenInteractionType(open) {
  const [openMethod, setOpenMethod] = React.useState(null);
  const handleTriggerClick = useStableCallback((_, interactionType) => {
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
  } = useEnhancedClickHandler(handleTriggerClick);
  return React.useMemo(() => ({
    openMethod,
    reset,
    triggerProps: {
      onClick,
      onPointerDown
    }
  }), [openMethod, reset, onClick, onPointerDown]);
}