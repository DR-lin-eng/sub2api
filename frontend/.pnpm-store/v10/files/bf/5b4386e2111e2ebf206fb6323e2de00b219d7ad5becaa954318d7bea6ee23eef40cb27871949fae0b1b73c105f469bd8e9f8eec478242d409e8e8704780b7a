//#region src/utils/composeEventHandlers.ts
/**
* Composes two event handlers together, calling the external handler first,
* then the internal handler if the event hasn't been prevented.
*
* @template E - The event type
* @param theirHandler - The external event handler (may be undefined)
* @param ourHandler - The internal event handler
* @returns A composed event handler that calls both handlers in sequence
*
* @example
* ```tsx
* <button
*   onClick={composeEventHandlers(externalOnClick, (e) => {
*     // Internal handler logic
*   })}
* />
* ```
*/
const composeEventHandlers = (theirHandler, ourHandler) => (event) => {
	theirHandler?.(event);
	if (event?.defaultPrevented) return;
	ourHandler(event);
};

//#endregion
export { composeEventHandlers };
//# sourceMappingURL=composeEventHandlers.mjs.map