/**
 * Moves an element in an array from one position to another.
 * @param {T[]} array
 * @param {number} from
 * @param {number} to
 * @returns {T[]}
 */
export declare function moveArrayElement<T>(array: T[], from: number, to: number): T[];
/**
 * Convert a hyphen-delimited string to camelCase.
 * @param {string} str
 * @returns {string}
 */
export declare function camelize(str: string): string;
/**
 * Convert an object's keys from hyphen-delimited to camelCase.
 * @param {Record<string, any>} object
 * @returns {Record<string, any>}
 */
export declare function objectMap(object: Record<any, any>): Record<string, any>;
/**
 * Removes an element from an array.
 * @param {T[]} array
 * @param {number} index
 * @returns {T[]}
 */
export declare function removeElement<T>(array: T[], index: number): T[];
/**
 * Inserts an element into an array.
 * @param {T[]} array
 * @param {number} index
 * @param element
 * @returns {T[]}
 */
export declare function insertElement<T>(array: T[], index: number, element: any): T[];
/**
 * If the value is undefined, return true, otherwise return false.
 * @param {any} value - any
 * @returns {value is undefined}
 */
export declare function isUndefined(value: any): value is undefined;
/**
 * If the value is string, return true, otherwise return false.
 * @param value
 * @returns {value is string}
 */
export declare function isString(value: any): value is string;
/**
 * Inserts a element into the DOM at a given index.
 * @param parentElement
 * @param element
 * @param {number} index
 */
export declare function insertNodeAt(parentElement: Element, element: Element, index: number): void;
/**
 * Removes a node from the DOM.
 * @param {Node} node
 */
export declare function removeNode(node: Node): void;
/**
 * Get an element by selector.
 * @param {string} selector
 * @param parentElement
 * @returns {Element}
 */
export declare function getElementBySelector(selector: string, parentElement?: Document | Element): HTMLElement;
/**
 * It takes a function and returns a function that executes the original function and then executes the second function.
 * @param {Function} fn - The function to be executed
 * @param {Function} afterFn - The function to be executed after the original function.
 * @param {any} [ctx=null] - The context of the function.
 * @returns {Function}
 */
export declare function mergeExecuted<T extends (...args: []) => any>(fn: T, afterFn: T, ctx?: any): (...args: any[]) => any;
/**
 * Merge the options and events.
 * @param {Record<string, any>} options
 * @param {Record<string, any>} events
 * @returns {Record<string, any>}
 */
export declare function mergeOptionsEvents(options: Record<string, any>, events: Record<string, any>): {
    [x: string]: any;
};
export declare function isHTMLElement(el: any): el is HTMLElement;
/**
 * @param obj
 * @param fn
 */
export declare function forEachObject<T extends Record<string, any>>(obj: T, fn: (key: keyof T, value: T[keyof T]) => void): void;
/**
 *
 * @param key
 */
export declare function isOn(key: any): boolean;
export declare const extend: {
    <T extends {}, U>(target: T, source: U): T & U;
    <T_1 extends {}, U_1, V>(target: T_1, source1: U_1, source2: V): T_1 & U_1 & V;
    <T_2 extends {}, U_2, V_1, W>(target: T_2, source1: U_2, source2: V_1, source3: W): T_2 & U_2 & V_1 & W;
    (target: object, ...sources: any[]): any;
};
