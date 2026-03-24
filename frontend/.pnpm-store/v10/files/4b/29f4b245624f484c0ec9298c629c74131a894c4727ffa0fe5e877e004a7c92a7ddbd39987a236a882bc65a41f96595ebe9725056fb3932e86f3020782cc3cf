import ActionIconGroup_default from "../../../ActionIconGroup/ActionIconGroup.mjs";
import { useChatListActionsBar } from "./useChatListActionsBar.mjs";
import { jsx } from "react/jsx-runtime";

//#region src/chat/ChatList/components/ChatActionsBar.tsx
const ChatActionsBar = ({ text, ref, ...rest }) => {
	const { regenerate, edit, copy, divider, del } = useChatListActionsBar(text);
	return /* @__PURE__ */ jsx(ActionIconGroup_default, {
		items: [regenerate, edit],
		menu: [
			edit,
			copy,
			regenerate,
			divider,
			del
		],
		ref,
		...rest
	});
};
var ChatActionsBar_default = ChatActionsBar;

//#endregion
export { ChatActionsBar_default as default };
//# sourceMappingURL=ChatActionsBar.mjs.map