'use client';

import usePreviewGroup_default from "./components/usePreviewGroup.mjs";
import { memo } from "react";
import { jsx } from "react/jsx-runtime";
import { Image } from "antd";

//#region src/Image/PreviewGroup.tsx
const { PreviewGroup: AntdPreviewGroup } = Image;
const PreviewGroup = memo(({ items, children, enable = true, preview }) => {
	const mergePreview = usePreviewGroup_default(preview);
	if (!enable) return children;
	return /* @__PURE__ */ jsx(AntdPreviewGroup, {
		items,
		preview: mergePreview,
		children
	});
});
PreviewGroup.displayName = "PreviewGroup";
var PreviewGroup_default = PreviewGroup;

//#endregion
export { PreviewGroup_default as default };
//# sourceMappingURL=PreviewGroup.mjs.map