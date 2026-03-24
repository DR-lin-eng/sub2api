import { copyToClipboard } from "../../../utils/copyToClipboard.mjs";
import ChatItem_default from "../../ChatItem/ChatItem.mjs";
import ChatActionsBar_default from "./ChatActionsBar.mjs";
import { memo, useCallback, useMemo, useState } from "react";
import { jsx } from "react/jsx-runtime";
import { App } from "antd";

//#region src/chat/ChatList/components/ChatListItem.tsx
const ChatListItem = memo((props) => {
	const { renderMessagesExtra, showTitle, onActionsClick, onAvatarsClick, onMessageChange, variant, text, renderMessages, renderErrorMessages, renderActions, loading, groupNav, renderItems, showAvatar, ...item } = props;
	const [editing, setEditing] = useState(false);
	const { message: message$1 } = App.useApp();
	const RenderItem = useMemo(() => {
		if (!renderItems || !item?.role) return;
		let renderFunction;
		if (renderItems?.[item.role]) renderFunction = renderItems[item.role];
		if (!renderFunction && renderItems?.["default"]) renderFunction = renderItems["default"];
		if (!renderFunction) return;
		return renderFunction;
	}, [renderItems?.[item.role]]);
	const RenderMessage = useCallback(({ editableContent, data }) => {
		if (!renderMessages || !item?.role) return;
		let RenderFunction;
		if (renderMessages?.[item.role]) RenderFunction = renderMessages[item.role];
		if (!RenderFunction && renderMessages?.["default"]) RenderFunction = renderMessages["default"];
		if (!RenderFunction) return;
		return /* @__PURE__ */ jsx(RenderFunction, {
			...data,
			editableContent
		});
	}, [renderMessages?.[item.role]]);
	const MessageExtra = useCallback(({ data }) => {
		if (!renderMessagesExtra || !item?.role) return;
		let RenderFunction;
		if (renderMessagesExtra?.[item.role]) RenderFunction = renderMessagesExtra[item.role];
		if (renderMessagesExtra?.["default"]) RenderFunction = renderMessagesExtra["default"];
		if (!RenderFunction) return;
		return /* @__PURE__ */ jsx(RenderFunction, { ...data });
	}, [renderMessagesExtra?.[item.role]]);
	const ErrorMessage = useCallback(({ data }) => {
		if (!renderErrorMessages || !item?.error?.type) return;
		let RenderFunction;
		if (renderErrorMessages?.[item.error.type]) RenderFunction = renderErrorMessages[item.error.type].Render;
		if (!RenderFunction && renderErrorMessages?.["default"]) RenderFunction = renderErrorMessages["default"].Render;
		if (!RenderFunction) return;
		return /* @__PURE__ */ jsx(RenderFunction, { ...data });
	}, [renderErrorMessages]);
	const Actions = useCallback(({ data }) => {
		if (!renderActions || !item?.role) return;
		let RenderFunction;
		if (renderActions?.[item.role]) RenderFunction = renderActions[item.role];
		if (renderActions?.["default"]) RenderFunction = renderActions["default"];
		if (!RenderFunction) RenderFunction = ChatActionsBar_default;
		const handleActionClick = async (action, data$1) => {
			switch (action.key) {
				case "copy":
					await copyToClipboard(data$1.content);
					message$1.success(text?.copySuccess || "Copy Success");
					break;
				case "edit": setEditing(true);
			}
			onActionsClick?.(action, data$1);
		};
		return /* @__PURE__ */ jsx(RenderFunction, {
			...data,
			onActionClick: (actionKey) => handleActionClick?.(actionKey, data),
			text
		});
	}, [
		renderActions?.[item.role],
		text,
		onActionsClick
	]);
	const error = useMemo(() => {
		if (!item.error) return;
		const message$2 = item.error?.message;
		let alertConfig = {};
		if (item.error.type && renderErrorMessages?.[item.error.type]) alertConfig = renderErrorMessages[item.error.type]?.config;
		return {
			message: message$2,
			...alertConfig
		};
	}, [renderErrorMessages, item.error]);
	if (RenderItem) return /* @__PURE__ */ jsx(RenderItem, { ...props }, item.id);
	return /* @__PURE__ */ jsx(ChatItem_default, {
		actions: /* @__PURE__ */ jsx(Actions, { data: item }),
		avatar: item.meta,
		avatarAddon: groupNav,
		editing,
		error,
		errorMessage: /* @__PURE__ */ jsx(ErrorMessage, { data: item }),
		loading,
		message: item.content,
		messageExtra: /* @__PURE__ */ jsx(MessageExtra, { data: item }),
		onAvatarClick: onAvatarsClick?.(item.role),
		onChange: (value) => onMessageChange?.(item.id, value),
		onDoubleClick: (e) => {
			if (item.id === "default" || item.error) return;
			if (item.role && ["assistant", "user"].includes(item.role) && e.altKey) setEditing(true);
		},
		onEditingChange: setEditing,
		placement: variant === "bubble" ? item.role === "user" ? "right" : "left" : "left",
		primary: item.role === "user",
		renderMessage: (editableContent) => /* @__PURE__ */ jsx(RenderMessage, {
			data: item,
			editableContent
		}),
		showAvatar,
		showTitle,
		text,
		time: item.updateAt || item.createAt,
		variant
	});
});
ChatListItem.displayName = "ChatListItem";
var ChatListItem_default = ChatListItem;

//#endregion
export { ChatListItem_default as default };
//# sourceMappingURL=ChatListItem.mjs.map