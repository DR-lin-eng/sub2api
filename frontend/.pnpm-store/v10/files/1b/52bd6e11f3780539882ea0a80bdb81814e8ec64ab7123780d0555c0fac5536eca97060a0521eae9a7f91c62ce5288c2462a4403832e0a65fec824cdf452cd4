import * as React from 'react';
import { type BaseUIChangeEventDetails } from "../../utils/createBaseUIEventDetails.js";
import { REASONS } from "../../utils/reasons.js";
/**
 * Groups all parts of the preview card.
 * Doesn’t render its own HTML element.
 *
 * Documentation: [Base UI Preview Card](https://base-ui.com/react/components/preview-card)
 */
export declare function PreviewCardRoot(props: PreviewCardRoot.Props): import("react/jsx-runtime").JSX.Element;
export interface PreviewCardRootState {}
export interface PreviewCardRootProps {
  children?: React.ReactNode;
  /**
   * Whether the preview card is initially open.
   *
   * To render a controlled preview card, use the `open` prop instead.
   * @default false
   */
  defaultOpen?: boolean;
  /**
   * Whether the preview card is currently open.
   */
  open?: boolean;
  /**
   * Event handler called when the preview card is opened or closed.
   */
  onOpenChange?: (open: boolean, eventDetails: PreviewCardRoot.ChangeEventDetails) => void;
  /**
   * Event handler called after any animations complete when the preview card is opened or closed.
   */
  onOpenChangeComplete?: (open: boolean) => void;
  /**
   * A ref to imperative actions.
   * - `unmount`: When specified, the preview card will not be unmounted when closed.
   * Instead, the `unmount` function must be called to unmount the preview card manually.
   * Useful when the preview card's animation is controlled by an external library.
   */
  actionsRef?: React.RefObject<PreviewCardRoot.Actions>;
}
export interface PreviewCardRootActions {
  unmount: () => void;
}
export type PreviewCardRootChangeEventReason = typeof REASONS.triggerHover | typeof REASONS.triggerFocus | typeof REASONS.triggerPress | typeof REASONS.outsidePress | typeof REASONS.escapeKey | typeof REASONS.none;
export type PreviewCardRootChangeEventDetails = BaseUIChangeEventDetails<PreviewCardRoot.ChangeEventReason>;
export declare namespace PreviewCardRoot {
  type State = PreviewCardRootState;
  type Props = PreviewCardRootProps;
  type Actions = PreviewCardRootActions;
  type ChangeEventReason = PreviewCardRootChangeEventReason;
  type ChangeEventDetails = PreviewCardRootChangeEventDetails;
}