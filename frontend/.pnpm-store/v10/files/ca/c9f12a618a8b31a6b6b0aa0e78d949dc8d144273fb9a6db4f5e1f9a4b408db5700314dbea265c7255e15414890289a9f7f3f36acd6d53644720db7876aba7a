'use client';

import FlexBasic_default from "../Flex/FlexBasic.mjs";
import SkeletonBlock_default from "./SkeletonBlock.mjs";
import { jsx } from "react/jsx-runtime";

//#region src/Skeleton/SkeletonParagraph.tsx
const DEFAULT_ROWS = 3;
const SkeletonParagraph = ({ active, rows = DEFAULT_ROWS, gap, width, height, fontSize, lineHeight, style, className, ...rest }) => {
	const rowGap = gap;
	const rowCount = Math.max(rows, 1);
	const baseFontSize = fontSize ?? 14;
	const resolvedLineHeight = lineHeight ?? 1.6;
	const marginBlock = (height ? height : Math.round(baseFontSize * resolvedLineHeight)) - baseFontSize;
	const widthList = Array.isArray(width) ? width : null;
	const getRowWidth = (index) => {
		if (widthList) return widthList[index] ?? widthList.at(-1) ?? "100%";
		if (width !== void 0) return width;
		if (index === rowCount - 1) return "66%";
		return "100%";
	};
	const containerStyle = {
		gap: rowGap,
		...style
	};
	return /* @__PURE__ */ jsx(FlexBasic_default, {
		className,
		gap: gap || marginBlock,
		style: containerStyle,
		width: "100%",
		...rest,
		children: Array.from({ length: rowCount }).map((_, index) => /* @__PURE__ */ jsx(SkeletonBlock_default, {
			active,
			height: baseFontSize,
			width: getRowWidth(index)
		}, index))
	});
};
SkeletonParagraph.displayName = "SkeletonParagraph";
var SkeletonParagraph_default = SkeletonParagraph;

//#endregion
export { SkeletonParagraph_default as default };
//# sourceMappingURL=SkeletonParagraph.mjs.map