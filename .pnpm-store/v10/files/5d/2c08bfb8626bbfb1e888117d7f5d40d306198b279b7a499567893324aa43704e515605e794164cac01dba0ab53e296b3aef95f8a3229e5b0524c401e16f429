import Sortable, { type Options, type SortableEvent } from 'sortablejs';
import { type Ref } from 'vue';
import type { RefOrElement, MaybeRef } from './types';
declare const CLONE_ELEMENT_KEY: unique symbol;
export interface DraggableEvent<T = any> extends SortableEvent {
    item: HTMLElement & {
        [CLONE_ELEMENT_KEY]: any;
    };
    data: T;
    clonedData: T;
}
type SortableMethod = 'closest' | 'save' | 'toArray' | 'destroy' | 'option';
export interface UseDraggableReturn extends Pick<Sortable, SortableMethod> {
    /**
     * Start the sortable.
     * @param {HTMLElement} target - The target element to be sorted.
     * @default By default the root element of the VueDraggablePlus instance is used
     */
    start: (target?: HTMLElement) => void;
    pause: () => void;
    resume: () => void;
}
export interface UseDraggableOptions<T> extends Options {
    clone?: (element: T) => T;
    immediate?: boolean;
    customUpdate?: (event: DraggableEvent<T>) => void;
    /**
     * Element dragging started
     */
    onStart?: ((event: DraggableEvent<T>) => void) | undefined;
    /**
     * Element dragging ended
     */
    onEnd?: ((event: DraggableEvent<T>) => void) | undefined;
    /**
     * Element is dropped into the list from another list
     */
    onAdd?: ((event: DraggableEvent<T>) => void) | undefined;
    /**
     * Created a clone of an element
     */
    onClone?: ((event: DraggableEvent<T>) => void) | undefined;
    /**
     * Element is chosen
     */
    onChoose?: ((event: DraggableEvent<T>) => void) | undefined;
    /**
     * Element is unchosen
     */
    onUnchoose?: ((event: DraggableEvent<T>) => void) | undefined;
    /**
     * Changed sorting within list
     */
    onUpdate?: ((event: DraggableEvent<T>) => void) | undefined;
    /**
     * Called by any change to the list (add / update / remove)
     */
    onSort?: ((event: DraggableEvent<T>) => void) | undefined;
    /**
     * Element is removed from the list into another list
     */
    onRemove?: ((event: DraggableEvent<T>) => void) | undefined;
    /**
     * Attempt to drag a filtered element
     */
    onFilter?: ((event: DraggableEvent<T>) => void) | undefined;
    /**
     * Called when dragging element changes position
     */
    onChange?: ((evt: DraggableEvent<T>) => void) | undefined;
}
/**
 * A custom compositionApi utils that allows you to drag and drop elements in lists.
 * @param el
 * @param {Array} list - The list to be dragged
 * @param {Object} options - The options of the sortable
 * @returns {Object} - The return of the sortable
 */
export declare function useDraggable<T>(el: RefOrElement, list?: Ref<T[] | undefined>, options?: MaybeRef<UseDraggableOptions<T>>): UseDraggableReturn;
export declare function useDraggable<T>(el: null | undefined, list?: Ref<T[] | undefined>, options?: MaybeRef<UseDraggableOptions<T>>): UseDraggableReturn;
export declare function useDraggable<T>(el: RefOrElement<HTMLElement | null | undefined>, options?: MaybeRef<UseDraggableOptions<T>>): UseDraggableReturn;
export {};
