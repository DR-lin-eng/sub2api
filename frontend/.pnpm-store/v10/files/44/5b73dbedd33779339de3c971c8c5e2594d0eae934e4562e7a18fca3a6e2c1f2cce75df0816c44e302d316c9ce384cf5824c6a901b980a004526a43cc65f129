import * as React from 'react';
import type { BaseUIComponentProps, NativeButtonProps, NonNativeButtonProps } from "../utils/types.js";
/**
 * A button component that can be used to trigger actions.
 * Renders a `<button>` element.
 *
 * Documentation: [Base UI Button](https://base-ui.com/react/components/button)
 */
export declare const Button: React.ForwardRefExoticComponent<ButtonProps & React.RefAttributes<HTMLElement>>;
export interface ButtonState {
  /**
   * Whether the button should ignore user interaction.
   */
  disabled: boolean;
}
interface ButtonCommonProps {
  /**
   * Whether the button should ignore user interaction.
   */
  disabled?: boolean;
  /**
   * Whether the button should be focusable when disabled.
   * @default false
   */
  focusableWhenDisabled?: boolean;
}
type NonNativeAttributeKeys = 'form' | 'formAction' | 'formEncType' | 'formMethod' | 'formNoValidate' | 'formTarget' | 'name' | 'type' | 'value';
interface ButtonNativeProps extends NativeButtonProps, ButtonCommonProps, Omit<BaseUIComponentProps<'button', ButtonState>, 'disabled'> {
  nativeButton?: true;
}
interface ButtonNonNativeProps extends NonNativeButtonProps, ButtonCommonProps, Omit<BaseUIComponentProps<'button', ButtonState>, NonNativeAttributeKeys | 'disabled'> {
  nativeButton: false;
}
export type ButtonProps = ButtonNativeProps | ButtonNonNativeProps;
export declare namespace Button {
  type State = ButtonState;
  type Props = ButtonProps;
}
export {};