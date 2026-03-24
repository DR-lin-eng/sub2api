'use client';

import A_default from "../../A/index.mjs";
import { safeParseJSON } from "../../utils/safeParseJSON.mjs";
import Citation_default from "./Citation/index.mjs";
import { jsx } from "react/jsx-runtime";

//#region src/mdx/mdxComponents/Link.tsx
const Link = ({ href, target, citations, ...rest }) => {
	if (rest["data-footnote-ref"]) return /* @__PURE__ */ jsx(Citation_default, {
		citationDetail: safeParseJSON(rest["data-link"]),
		href,
		id: rest.id,
		inSup: true,
		children: rest.children
	});
	const match = href?.match(/citation-(\d+)/);
	if (match) {
		const index = Number.parseInt(match[1]) - 1;
		const detail = citations?.[index];
		return /* @__PURE__ */ jsx(Citation_default, {
			citationDetail: detail,
			id: match[1],
			children: match[1]
		});
	}
	const isNewWindow = href?.startsWith("http");
	return /* @__PURE__ */ jsx(A_default, {
		href,
		target: target || isNewWindow ? "_blank" : void 0,
		...rest
	});
};
Link.displayName = "MdxLink";
var Link_default = Link;

//#endregion
export { Link_default as default };
//# sourceMappingURL=Link.mjs.map