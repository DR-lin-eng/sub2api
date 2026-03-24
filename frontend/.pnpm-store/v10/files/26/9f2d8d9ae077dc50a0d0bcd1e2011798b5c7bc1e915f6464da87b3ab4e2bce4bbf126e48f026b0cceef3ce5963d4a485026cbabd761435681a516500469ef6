'use client';

import { FALLBACK_LANG } from "../../Highlighter/const.mjs";
import Highlighter_default from "../../Highlighter/Highlighter.mjs";
import Mermaid_default from "../../Mermaid/Mermaid.mjs";
import Snippet_default from "../../Snippet/Snippet.mjs";
import { jsx } from "react/jsx-runtime";
import { createStaticStyles, cx } from "antd-style";

//#region src/mdx/mdxComponents/Pre.tsx
const styles = createStaticStyles(({ css: css$1 }) => ({ container: css$1`
    overflow: hidden;
    margin-block: calc(var(--lobe-markdown-margin-multiple) * 0.5em);
    border-radius: calc(var(--lobe-markdown-border-radius) * 1px);
    box-shadow: 0 0 0 1px var(--lobe-markdown-border-color) inset;
  ` }));
const Pre = ({ fullFeatured, fileName, allowChangeLanguage, language = FALLBACK_LANG, children, className, style, variant = "filled", icon, theme, ...rest }) => {
	return /* @__PURE__ */ jsx(Highlighter_default, {
		allowChangeLanguage,
		className: cx(styles.container, className),
		fileName,
		fullFeatured,
		icon,
		language,
		style,
		theme,
		variant,
		...rest,
		children
	});
};
const PreSingleLine = ({ language = FALLBACK_LANG, children, className, style, variant = "filled", ...rest }) => {
	return /* @__PURE__ */ jsx(Snippet_default, {
		className: cx(styles.container, className),
		"data-code-type": "highlighter",
		language,
		style,
		variant,
		...rest,
		children
	});
};
const PreMermaid = ({ animated, fullFeatured, children, className, style, variant = "filled", theme, ...rest }) => {
	return /* @__PURE__ */ jsx(Mermaid_default, {
		animated,
		className: cx(styles.container, className),
		fullFeatured,
		style,
		theme,
		variant,
		...rest,
		children
	});
};
Pre.displayName = "MdxPre";
var Pre_default = Pre;

//#endregion
export { Pre, PreMermaid, PreSingleLine, Pre_default as default };
//# sourceMappingURL=Pre.mjs.map