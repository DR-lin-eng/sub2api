'use client';

import FlexBasic_default from "../Flex/FlexBasic.mjs";
import ActionIcon_default from "../ActionIcon/ActionIcon.mjs";
import Input_default from "../Input/Input.mjs";
import { memo, useEffect, useRef, useState } from "react";
import { jsx, jsxs } from "react/jsx-runtime";
import { RotateCcw, Save } from "lucide-react";

//#region src/EditableText/ControlInput.tsx
const ControlInput = memo(({ value, onChange, onValueChanging, onChangeEnd, onCompositionEnd, onCompositionStart, onPressEnter, onFocus, submitIcon, style, texts, ...rest }) => {
	const ref = useRef(null);
	const [input, setInput] = useState(value || "");
	const isChineseInput = useRef(false);
	useEffect(() => {
		if (value !== void 0) setInput(value);
	}, [value]);
	const handleUpload = () => {
		onChange?.(input);
		ref?.current?.blur();
		onChangeEnd?.(input);
	};
	return /* @__PURE__ */ jsx(Input_default, {
		autoFocus: true,
		onChange: (e) => {
			setInput(e.target.value);
			onValueChanging?.(e.target.value);
		},
		onCompositionEnd: (e) => {
			isChineseInput.current = false;
			onCompositionEnd?.(e);
		},
		onCompositionStart: (e) => {
			isChineseInput.current = true;
			onCompositionStart?.(e);
		},
		onFocus,
		onPressEnter: (e) => {
			if (!e.shiftKey && !isChineseInput.current) {
				e.preventDefault();
				handleUpload();
				onPressEnter?.(e);
			}
		},
		ref,
		style: {
			width: "100%",
			...style
		},
		suffix: value === input ? /* @__PURE__ */ jsx("span", {}) : /* @__PURE__ */ jsxs(FlexBasic_default, {
			gap: 2,
			horizontal: true,
			style: {
				marginRight: -4,
				zIndex: 1
			},
			children: [/* @__PURE__ */ jsx(ActionIcon_default, {
				icon: RotateCcw,
				onClick: (e) => {
					e.preventDefault();
					setInput(value || "");
				},
				size: "small",
				title: texts?.reset || "Reset"
			}), /* @__PURE__ */ jsx(ActionIcon_default, {
				icon: submitIcon || Save,
				onClick: (e) => {
					e.preventDefault();
					handleUpload();
				},
				size: "small",
				title: texts?.submit || "Submit",
				variant: "filled"
			})]
		}),
		value: input,
		...rest
	});
});
ControlInput.displayName = "ControlInput";
var ControlInput_default = ControlInput;

//#endregion
export { ControlInput_default as default };
//# sourceMappingURL=ControlInput.mjs.map