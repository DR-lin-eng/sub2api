'use client';

import TextArea_default from "../../../Input/TextArea.mjs";
import { memo, useRef } from "react";
import { jsx } from "react/jsx-runtime";

//#region src/chat/ChatInputArea/components/ChatInputAreaInner.tsx
const ChatInputAreaInner = memo(({ ref, resize = false, onCompositionEnd, onPressEnter, onCompositionStart, className, onInput, loading, onSend, onBlur, onChange, ...rest }) => {
	const isChineseInput = useRef(false);
	return /* @__PURE__ */ jsx(TextArea_default, {
		className,
		onBlur: (e) => {
			onInput?.(e.target.value);
			onBlur?.(e);
		},
		onChange: (e) => {
			onInput?.(e.target.value);
			onChange?.(e);
		},
		onCompositionEnd: (e) => {
			isChineseInput.current = false;
			onCompositionEnd?.(e);
		},
		onCompositionStart: (e) => {
			isChineseInput.current = true;
			onCompositionStart?.(e);
		},
		onPressEnter: (e) => {
			onPressEnter?.(e);
			const isMobile = /mobi|android|iphone/i.test(navigator.userAgent);
			if (!loading && !isChineseInput.current && (!isMobile && !e.shiftKey || isMobile && e.shiftKey)) {
				e.preventDefault();
				onSend?.();
			}
		},
		ref,
		resize,
		variant: "borderless",
		...rest
	});
});
var ChatInputAreaInner_default = ChatInputAreaInner;

//#endregion
export { ChatInputAreaInner_default as default };
//# sourceMappingURL=ChatInputAreaInner.mjs.map