'use client';

import { useMarkdownComponents } from "../../hooks/useMarkdown/useMarkdownComponents.mjs";
import { useMarkdownContent } from "../../hooks/useMarkdown/useMarkdownContent.mjs";
import { useMarkdownRehypePlugins } from "../../hooks/useMarkdown/useMarkdownRehypePlugins.mjs";
import { useMarkdownRemarkPlugins } from "../../hooks/useMarkdown/useMarkdownRemarkPlugins.mjs";
import { styles } from "./style.mjs";
import { createElement, memo, useId, useMemo } from "react";
import { jsx } from "react/jsx-runtime";
import Markdown from "react-markdown";
import { marked } from "marked";

//#region src/Markdown/SyntaxMarkdown/StreamdownRender.tsx
const parseMarkdownIntoBlocks = (markdown) => {
	return marked.lexer(markdown).map((token) => token.raw);
};
const StreamdownBlock = memo(({ children, ...rest }) => {
	return /* @__PURE__ */ jsx(Markdown, {
		...rest,
		children
	});
}, (prevProps, nextProps) => prevProps.children === nextProps.children);
StreamdownBlock.displayName = "StreamdownBlock";
const StreamdownRender = memo(({ children, ...rest }) => {
	const escapedContent = useMarkdownContent(children || "");
	const components = useMarkdownComponents();
	const rehypePluginsList = useMarkdownRehypePlugins();
	const remarkPluginsList = useMarkdownRemarkPlugins();
	const generatedId = useId();
	const blocks = useMemo(() => parseMarkdownIntoBlocks(typeof escapedContent === "string" ? escapedContent : ""), [escapedContent]);
	return /* @__PURE__ */ jsx("div", {
		className: styles.animated,
		children: blocks.map((block, index) => /* @__PURE__ */ createElement(StreamdownBlock, {
			...rest,
			components,
			key: `${generatedId}-block_${index}`,
			rehypePlugins: rehypePluginsList,
			remarkPlugins: remarkPluginsList
		}, block))
	});
}, (prevProps, nextProps) => prevProps.children === nextProps.children);
StreamdownRender.displayName = "StreamdownRender";
var StreamdownRender_default = StreamdownRender;

//#endregion
export { StreamdownRender_default as default };
//# sourceMappingURL=StreamdownRender.mjs.map