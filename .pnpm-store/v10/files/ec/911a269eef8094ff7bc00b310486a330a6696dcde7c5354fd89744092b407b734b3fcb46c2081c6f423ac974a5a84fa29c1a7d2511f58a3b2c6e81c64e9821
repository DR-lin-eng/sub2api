import * as React from 'react';
import { BaseUIComponentProps, NativeButtonProps } from "../../utils/types.js";
import type { FieldRoot } from "../../field/root/FieldRoot.js";
/**
 * A button that opens the select popup.
 * Renders a `<button>` element.
 *
 * Documentation: [Base UI Select](https://base-ui.com/react/components/select)
 */
export declare const SelectTrigger: React.ForwardRefExoticComponent<SelectTriggerProps & React.RefAttributes<HTMLButtonElement>>;
export interface SelectTriggerState extends FieldRoot.State {
  /** Whether the select popup is currently open. */
  open: boolean;
  /** Whether the select popup is readonly. */
  readOnly: boolean;
  /** The value of the currently selected item. */
  value: any;
}
export interface SelectTriggerProps extends NativeButtonProps, BaseUIComponentProps<'button', SelectTrigger.State> {
  children?: React.ReactNode;
  /** Whether the component should ignore user interaction. */
  disabled?: boolean;
}
export declare namespace SelectTrigger {
  type State = SelectTriggerState;
  type Props = SelectTriggerProps;
}