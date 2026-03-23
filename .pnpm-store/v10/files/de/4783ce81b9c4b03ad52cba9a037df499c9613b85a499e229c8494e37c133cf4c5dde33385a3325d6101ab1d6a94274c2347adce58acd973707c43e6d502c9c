'use client';

import ActionIcon_default from "../ActionIcon/ActionIcon.mjs";
import { styles, variants } from "./style.mjs";
import { jsx } from "react/jsx-runtime";
import { Tabs } from "antd";
import { cx } from "antd-style";
import { MoreHorizontalIcon } from "lucide-react";

//#region src/Tabs/Tabs.tsx
const Tabs$1 = ({ className, compact, variant = "rounded", items, ...rest }) => {
	const hasContent = items?.some((item) => !!item.children);
	return /* @__PURE__ */ jsx(Tabs, {
		className: cx(variants({
			compact,
			underlined: hasContent,
			variant
		}), className),
		items,
		...rest,
		classNames: {
			...rest?.classNames,
			popup: {
				root: styles.dropdown,
				...rest?.classNames?.popup
			}
		},
		more: {
			icon: /* @__PURE__ */ jsx(ActionIcon_default, { icon: MoreHorizontalIcon }),
			...rest?.more
		}
	});
};
Tabs$1.displayName = "Tabs";
var Tabs_default = Tabs$1;

//#endregion
export { Tabs_default as default };
//# sourceMappingURL=Tabs.mjs.map