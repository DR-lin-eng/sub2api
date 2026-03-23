//#region src/Tooltip/antdPlacementToFloating.ts
/**
* Convert Ant Design legacy placements to Floating UI placements.
*
* Notes:
* - Floating UI placements like `top-start` are passed through.
* - Ant Design legacy placements like `topLeft` are mapped to Floating UI.
*/
const antdPlacementToFloating = (placement) => {
	if (!placement) return "top";
	if (typeof placement === "string" && placement.includes("-")) return placement;
	switch (placement) {
		case "topLeft": return "top-start";
		case "topRight": return "top-end";
		case "bottomLeft": return "bottom-start";
		case "bottomRight": return "bottom-end";
		case "leftTop": return "left-start";
		case "leftBottom": return "left-end";
		case "rightTop": return "right-start";
		case "rightBottom": return "right-end";
		default: return placement;
	}
};

//#endregion
export { antdPlacementToFloating };
//# sourceMappingURL=antdPlacementToFloating.mjs.map