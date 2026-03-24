//#region src/Flex/utils.ts
const getFlexDirection = (direction, isHorizontal$1) => {
	if (isHorizontal$1) return "row";
	switch (direction) {
		case "horizontal": return "row";
		case "horizontal-reverse": return "row-reverse";
		case "vertical":
		default: return "column";
		case "vertical-reverse": return "column-reverse";
	}
};
const isSpaceDistribution = (distribution) => {
	if (!distribution) return;
	return [
		"space-between",
		"space-around",
		"space-evenly"
	].includes(distribution);
};
const isHorizontal = (direction, isHorizontal$1) => getFlexDirection(direction, isHorizontal$1) === "row";
const getCssValue = (value) => typeof value === "number" ? `${value}px` : value;

//#endregion
export { getCssValue, getFlexDirection, isHorizontal, isSpaceDistribution };
//# sourceMappingURL=utils.mjs.map