'use client';

import StaticMermaid_default from "./StaticMermaid.mjs";
import StreamMermaid_default from "./StreamMermaid.mjs";
import { variants } from "./style.mjs";
import { memo } from "react";
import { jsx } from "react/jsx-runtime";
import { cx } from "antd-style";

//#region src/Mermaid/SyntaxMermaid/index.tsx
const SyntaxMermaid = memo(({ animated, children, className, fallbackClassName, ref, style, theme: customTheme, variant = "borderless" }) => {
	const isDefaultTheme = customTheme === "lobe-theme" || !customTheme;
	const showBackground = !isDefaultTheme && variant === "filled";
	const resolvedTheme = isDefaultTheme ? void 0 : customTheme;
	const mermaidClassName = cx(variants({
		animated,
		mermaid: true,
		showBackground,
		variant
	}), className);
	const fallback = cx(variants({
		animated,
		mermaid: false,
		showBackground,
		variant
	}), fallbackClassName);
	if (animated) return /* @__PURE__ */ jsx(StreamMermaid_default, {
		className: mermaidClassName,
		fallbackClassName: fallback,
		ref,
		style,
		theme: resolvedTheme,
		variant,
		children
	});
	return /* @__PURE__ */ jsx(StaticMermaid_default, {
		className: mermaidClassName,
		fallbackClassName: fallback,
		ref,
		style,
		theme: resolvedTheme,
		variant,
		children
	});
}, (prevProps, nextProps) => prevProps.children === nextProps.children && prevProps.animated === nextProps.animated);
SyntaxMermaid.displayName = "SyntaxMermaid";
var SyntaxMermaid_default = SyntaxMermaid;

//#endregion
export { SyntaxMermaid_default as default };
//# sourceMappingURL=index.mjs.map