'use client';

import ScrollShadow_default from "../../../ScrollShadow/ScrollShadow.mjs";
import SearchResultCard_default from "./SearchResultCard.mjs";
import { memo } from "react";
import { jsx } from "react/jsx-runtime";

//#region src/Markdown/components/SearchResultCards/index.tsx
const SearchResultCards = memo(({ ref, dataSource, style, ...rest }) => {
	return /* @__PURE__ */ jsx(ScrollShadow_default, {
		gap: 12,
		hideScrollBar: true,
		horizontal: true,
		orientation: "horizontal",
		ref,
		size: 10,
		style: {
			minHeight: 80,
			overflowX: "scroll",
			paddingInlineEnd: 24,
			width: "100%",
			...style
		},
		...rest,
		children: dataSource.map((link) => typeof link === "string" ? /* @__PURE__ */ jsx(SearchResultCard_default, { url: link }, link) : /* @__PURE__ */ jsx(SearchResultCard_default, { ...link }, link.url))
	});
});
SearchResultCards.displayName = "SearchResultCards";
var SearchResultCards_default = SearchResultCards;

//#endregion
export { SearchResultCards_default as default };
//# sourceMappingURL=index.mjs.map