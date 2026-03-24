import Icon_default from "../../../Icon/Icon.mjs";
import Tag_default from "../../../Tag/Tag.mjs";
import { jsx } from "react/jsx-runtime";
import { Divider } from "antd";
import { Timer } from "lucide-react";

//#region src/chat/ChatList/components/HistoryDivider.tsx
const HistoryDivider = ({ enable, text }) => {
	if (!enable) return null;
	return /* @__PURE__ */ jsx("div", {
		style: { padding: "0 20px" },
		children: /* @__PURE__ */ jsx(Divider, { children: /* @__PURE__ */ jsx(Tag_default, {
			icon: /* @__PURE__ */ jsx(Icon_default, { icon: Timer }),
			children: text || "History Message"
		}) })
	});
};
var HistoryDivider_default = HistoryDivider;

//#endregion
export { HistoryDivider_default as default };
//# sourceMappingURL=HistoryDivider.mjs.map