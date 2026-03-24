import Icon_default from "../../Icon/Icon.mjs";
import { styles } from "../style.mjs";
import Preview_default from "./Preview.mjs";
import Toolbar_default from "./Toolbar.mjs";
import { useMemo, useState } from "react";
import { jsx } from "react/jsx-runtime";
import { cx } from "antd-style";
import { X } from "lucide-react";

//#region src/Image/components/usePreviewGroup.tsx
const usePreview = (props) => {
	const [visible, setVisible] = useState(false);
	return useMemo(() => {
		if (props === false) return props;
		const { onVisibleChange, onOpenChange, minScale = .32, maxScale = 32, toolbarAddon, rootClassName, imageRender, toolbarRender, ...rest } = props === true ? {} : props || {};
		return {
			actionsRender: (_, info) => {
				const originalNode = /* @__PURE__ */ jsx(Toolbar_default, {
					info,
					maxScale,
					minScale,
					children: toolbarAddon
				});
				if (toolbarRender) return toolbarRender(originalNode, info);
				return originalNode;
			},
			closeIcon: /* @__PURE__ */ jsx(Icon_default, {
				color: "#fff",
				icon: X
			}),
			imageRender: (originalNode, info) => {
				const node = /* @__PURE__ */ jsx(Preview_default, {
					visible,
					children: originalNode
				});
				if (imageRender) return imageRender(node, info);
				return node;
			},
			maxScale,
			minScale,
			onOpenChange: (open, info) => {
				setVisible(open);
				onOpenChange?.(open, info);
				onVisibleChange?.(open, !open, info.current);
			},
			rootClassName: cx(styles.preview, rootClassName),
			...rest
		};
	}, [
		props,
		visible,
		styles
	]);
};
var usePreviewGroup_default = usePreview;

//#endregion
export { usePreviewGroup_default as default };
//# sourceMappingURL=usePreviewGroup.mjs.map