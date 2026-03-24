import * as React from 'react';
import { BaseUIComponentProps, NativeButtonProps } from "../../utils/types.js";
import type { FieldRoot } from "../../field/root/FieldRoot.js";
/**
 * A button that opens the popup.
 * Renders a `<button>` element.
 */
export declare const ComboboxTrigger: React.ForwardRefExoticComponent<ComboboxTriggerProps & React.RefAttributes<HTMLButtonElement>>;
export interface ComboboxTriggerState extends FieldRoot.State {
  /**
   * Whether the popup is open.
   */
  open: boolean;
  /**
   * Whether the component should ignore user interaction.
   */
  disabled: boolean;
}
export interface ComboboxTriggerProps extends NativeButtonProps, BaseUIComponentProps<'button', ComboboxTrigger.State> {
  /**
   * Whether the component should ignore user interaction.
   * @default false
   */
  disabled?: boolean;
}
export declare namespace ComboboxTrigger {
  type State = ComboboxTriggerState;
  type Props = ComboboxTriggerProps;
}