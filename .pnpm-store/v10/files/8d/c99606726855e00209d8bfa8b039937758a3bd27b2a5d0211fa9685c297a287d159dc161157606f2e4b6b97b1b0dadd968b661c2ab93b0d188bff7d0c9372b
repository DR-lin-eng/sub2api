'use client';

import FlexBasic_default from "../../Flex/FlexBasic.mjs";
import GridBackground_default from "./GridBackground.mjs";
import { memo } from "react";
import { jsx, jsxs } from "react/jsx-runtime";
import { useTheme } from "antd-style";
import { rgba } from "polished";

//#region src/awesome/GridBackground/GridShowcase.tsx
const GridShowcase = memo(({ style, children, backgroundColor = "#001dff", innerProps, ...rest }) => {
	const theme = useTheme();
	return /* @__PURE__ */ jsxs(FlexBasic_default, {
		style: {
			position: "relative",
			...style
		},
		...rest,
		children: [
			/* @__PURE__ */ jsx(GridBackground_default, {
				animation: true,
				colorBack: rgba(theme.colorText, .12),
				colorFront: rgba(theme.colorText, .6),
				flip: true
			}),
			/* @__PURE__ */ jsx(FlexBasic_default, {
				align: "center",
				...innerProps,
				style: {
					zIndex: 4,
					...innerProps?.style
				},
				children
			}),
			/* @__PURE__ */ jsx(GridBackground_default, {
				animation: true,
				backgroundColor,
				colorBack: rgba(theme.colorText, .24),
				colorFront: theme.colorText,
				random: true,
				reverse: true,
				showBackground: true,
				style: { zIndex: 0 }
			})
		]
	});
});
GridShowcase.displayName = "GridShowcase";
var GridShowcase_default = GridShowcase;

//#endregion
export { GridShowcase_default as default };
//# sourceMappingURL=GridShowcase.mjs.map