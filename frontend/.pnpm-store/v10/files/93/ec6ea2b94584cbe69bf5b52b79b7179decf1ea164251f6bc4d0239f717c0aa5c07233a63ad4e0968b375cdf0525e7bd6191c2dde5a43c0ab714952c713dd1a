import * as React from 'react';
import type { BaseUIEvent, WithBaseUIEvent } from "../utils/types.js";
type ElementType = React.ElementType;
type PropsOf<T extends React.ElementType> = WithBaseUIEvent<React.ComponentPropsWithRef<T>>;
type InputProps<T extends React.ElementType> = PropsOf<T> | ((otherProps: PropsOf<T>) => PropsOf<T>) | undefined;
/**
 * Merges multiple sets of React props. It follows the Object.assign pattern where the rightmost object's fields overwrite
 * the conflicting ones from others. This doesn't apply to event handlers, `className` and `style` props.
 * Event handlers are merged such that they are called in sequence (the rightmost one being called first),
 * and allows the user to prevent the subsequent event handlers from being
 * executed by attaching a `preventBaseUIHandler` method.
 * It also merges the `className` and `style` props, whereby the classes are concatenated
 * and the rightmost styles overwrite the subsequent ones.
 *
 * Props can either be provided as objects or as functions that take the previous props as an argument.
 * The function will receive the merged props up to that point (going from left to right):
 * so in the case of `(obj1, obj2, fn, obj3)`, `fn` will receive the merged props of `obj1` and `obj2`.
 * The function is responsible for chaining event handlers if needed (i.e. we don't run the merge logic).
 *
 * Event handlers returned by the functions are not automatically prevented when `preventBaseUIHandler` is called.
 * They must check `event.baseUIHandlerPrevented` themselves and bail out if it's true.
 *
 * @important **`ref` is not merged.**
 * @param props props to merge.
 * @returns the merged props.
 */
export declare function mergeProps<T extends ElementType>(a: InputProps<T>, b: InputProps<T>): PropsOf<T>;
export declare function mergeProps<T extends ElementType>(a: InputProps<T>, b: InputProps<T>): PropsOf<T>;
export declare function mergeProps<T extends ElementType>(a: InputProps<T>, b: InputProps<T>, c: InputProps<T>): PropsOf<T>;
export declare function mergeProps<T extends ElementType>(a: InputProps<T>, b: InputProps<T>, c: InputProps<T>, d: InputProps<T>): PropsOf<T>;
export declare function mergeProps<T extends ElementType>(a: InputProps<T>, b: InputProps<T>, c: InputProps<T>, d: InputProps<T>, e: InputProps<T>): PropsOf<T>;
export declare function mergePropsN<T extends ElementType>(props: InputProps<T>[]): PropsOf<T>;
export declare function makeEventPreventable<T extends React.SyntheticEvent>(event: BaseUIEvent<T>): BaseUIEvent<T>;
export declare function mergeClassNames(ourClassName: string | undefined, theirClassName: string | undefined): string | undefined;
export {};