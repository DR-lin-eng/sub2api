'use client';

import { createContext, memo, use } from "react";
import { jsx } from "react/jsx-runtime";

//#region src/Markdown/components/MarkdownProvider.tsx
const MarkdownContext = createContext({});
const MarkdownProvider = memo(({ children, ...config }) => {
	return /* @__PURE__ */ jsx(MarkdownContext, {
		value: config,
		children
	});
});
const useMarkdownContext = () => {
	return use(MarkdownContext);
};

//#endregion
export { MarkdownProvider, useMarkdownContext };
//# sourceMappingURL=MarkdownProvider.mjs.map