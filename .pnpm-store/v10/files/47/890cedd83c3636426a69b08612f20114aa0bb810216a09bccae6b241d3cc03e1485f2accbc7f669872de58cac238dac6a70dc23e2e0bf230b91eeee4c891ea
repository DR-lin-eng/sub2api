'use client';

import SkeletonBlock_default from "./SkeletonBlock.mjs";
import { jsx } from "react/jsx-runtime";
import { cssVar } from "antd-style";

//#region src/Skeleton/SkeletonButton.tsx
const HEIGHT_MAP = {
	default: 36,
	large: 46,
	small: 28
};
const SkeletonButton = ({ active, block = false, shape = "default", size = "default", width, height, style, className, ...rest }) => {
	const resolvedSize = size ?? "default";
	const baseHeight = height ?? HEIGHT_MAP[resolvedSize];
	const finalWidth = width ?? (block ? "100%" : shape === "circle" ? baseHeight : 80);
	const RADIUS_MAP = {
		default: cssVar.borderRadius,
		large: cssVar.borderRadiusLG,
		small: cssVar.borderRadiusSM
	};
	return /* @__PURE__ */ jsx(SkeletonBlock_default, {
		active,
		className,
		height: baseHeight,
		style: {
			borderRadius: shape === "circle" ? "50%" : shape === "round" ? `calc(${cssVar.borderRadius} * 2)` : RADIUS_MAP[resolvedSize],
			...style
		},
		width: finalWidth,
		...rest
	});
};
SkeletonButton.displayName = "SkeletonButton";
var SkeletonButton_default = SkeletonButton;

//#endregion
export { SkeletonButton_default as default };
//# sourceMappingURL=SkeletonButton.mjs.map