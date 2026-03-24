import * as React from 'react';
import type { FloatingRootContext } from "../../floating-ui-react/index.js";
import type { TransitionStatus } from "../../utils/useTransitionStatus.js";
import type { HTMLProps } from "../../utils/types.js";
import type { PreviewCardRoot } from "./PreviewCardRoot.js";
export interface PreviewCardTriggerDelayConfig {
  delay?: number;
  closeDelay?: number;
}
export interface PreviewCardRootContext {
  open: boolean;
  setOpen: (open: boolean, eventDetails: PreviewCardRoot.ChangeEventDetails) => void;
  setTriggerElement: (el: Element | null) => void;
  positionerElement: HTMLElement | null;
  setPositionerElement: (el: HTMLElement | null) => void;
  mounted: boolean;
  setMounted: React.Dispatch<React.SetStateAction<boolean>>;
  triggerProps: HTMLProps;
  popupProps: HTMLProps;
  floatingRootContext: FloatingRootContext;
  transitionStatus: TransitionStatus;
  popupRef: React.RefObject<HTMLElement | null>;
  onOpenChangeComplete: ((open: boolean) => void) | undefined;
  writeDelayRefs: (config: PreviewCardTriggerDelayConfig) => void;
}
export declare const PreviewCardRootContext: React.Context<PreviewCardRootContext | undefined>;
export declare function usePreviewCardRootContext(): PreviewCardRootContext;