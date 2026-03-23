'use client';

import { styles } from "../style.mjs";
import { memo } from "react";
import { jsx } from "react/jsx-runtime";
import { cx } from "antd-style";

//#region src/Layout/components/LayoutHeader.tsx
const LayoutHeader = memo(({ headerHeight, children, className, style, ...rest }) => {
	return /* @__PURE__ */ jsx("header", {
		className: cx(styles.header, className),
		style: {
			height: headerHeight,
			...style
		},
		...rest,
		children
	});
});
LayoutHeader.displayName = "LayoutHeader";
var LayoutHeader_default = LayoutHeader;

//#endregion
export { LayoutHeader_default as default };
//# sourceMappingURL=LayoutHeader.mjs.map