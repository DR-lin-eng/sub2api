'use client';

import SpotlightCard_default from "../SpotlightCard/SpotlightCard.mjs";
import FeatureItem_default from "./FeatureItem.mjs";
import { memo } from "react";
import { jsx } from "react/jsx-runtime";

//#region src/awesome/Features/Features.tsx
const Features = memo(({ items, className, itemClassName, itemStyle, maxWidth = 960, style, ...rest }) => {
	if (!items?.length) return;
	return /* @__PURE__ */ jsx(SpotlightCard_default, {
		className,
		items,
		renderItem: (item) => /* @__PURE__ */ jsx(FeatureItem_default, {
			className: itemClassName,
			style: itemStyle,
			...item
		}, item.title),
		style: {
			maxWidth,
			...style
		},
		...rest
	});
});
Features.displayName = "Features";
var Features_default = Features;

//#endregion
export { Features_default as default };
//# sourceMappingURL=Features.mjs.map