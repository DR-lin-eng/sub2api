'use client';

import Footnotes_default from "../../Markdown/components/Footnotes.mjs";
import { jsx } from "react/jsx-runtime";

//#region src/mdx/mdxComponents/Section.tsx
const Section = ({ children, showFootnotes, ...rest }) => {
	if (rest["data-footnotes"]) {
		if (!showFootnotes) return null;
		return /* @__PURE__ */ jsx(Footnotes_default, {
			...rest,
			children
		});
	}
	return /* @__PURE__ */ jsx("section", {
		...rest,
		children
	});
};
Section.displayName = "MdxSection";
var Section_default = Section;

//#endregion
export { Section_default as default };
//# sourceMappingURL=Section.mjs.map