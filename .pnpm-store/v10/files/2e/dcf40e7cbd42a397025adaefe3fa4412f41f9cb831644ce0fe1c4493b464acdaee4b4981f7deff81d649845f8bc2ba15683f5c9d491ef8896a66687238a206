'use client';

import ChatListItem_default from "./components/ChatListItem.mjs";
import HistoryDivider_default from "./components/HistoryDivider.mjs";
import { styles } from "./style.mjs";
import { Fragment, memo } from "react";
import { jsx, jsxs } from "react/jsx-runtime";
import { cx } from "antd-style";

//#region src/chat/ChatList/ChatList.tsx
const ChatList = memo(({ onActionsClick, onAvatarsClick, renderMessagesExtra, className, data, variant = "bubble", text, showTitle, onMessageChange, renderMessages, renderErrorMessages, loadingId, renderItems, enableHistoryCount, renderActions, historyCount = 0, showAvatar, ...rest }) => {
	return /* @__PURE__ */ jsx("div", {
		className: cx(styles.container, className),
		...rest,
		children: data.map((item, index) => {
			const itemProps = {
				loading: loadingId === item.id,
				onActionsClick,
				onAvatarsClick,
				onMessageChange,
				renderActions,
				renderErrorMessages,
				renderItems,
				renderMessages,
				renderMessagesExtra,
				showAvatar,
				showTitle,
				text,
				variant
			};
			const historyLength = data.length;
			return /* @__PURE__ */ jsxs(Fragment, { children: [/* @__PURE__ */ jsx(HistoryDivider_default, {
				enable: enableHistoryCount && historyLength > historyCount && historyCount === historyLength - index + 1,
				text: text?.history
			}), /* @__PURE__ */ jsx(ChatListItem_default, {
				...itemProps,
				...item
			})] }, item.id);
		})
	});
});
var ChatList_default = ChatList;

//#endregion
export { ChatList_default as default };
//# sourceMappingURL=ChatList.mjs.map