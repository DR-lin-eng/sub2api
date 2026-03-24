'use client';

import ActionIcon_default from "../../ActionIcon/ActionIcon.mjs";
import { SortableItemContext } from "./SortableItem.mjs";
import { memo, use, useState } from "react";
import { jsx } from "react/jsx-runtime";
import { GripVertical } from "lucide-react";

//#region src/SortableList/components/DragHandle.tsx
const DragHandle = memo(({ style, ...rest }) => {
	const [grab, setGrab] = useState(false);
	const { attributes, listeners, ref } = use(SortableItemContext);
	return /* @__PURE__ */ jsx(ActionIcon_default, {
		"data-cypress": "draggable-handle",
		glass: true,
		icon: GripVertical,
		onMouseDown: () => setGrab(true),
		onMouseUp: () => setGrab(false),
		size: "small",
		style: {
			cursor: grab ? "grab" : "grabbing",
			...style
		},
		...rest,
		...attributes,
		...listeners,
		ref
	});
});
DragHandle.displayName = "DragHandle";
var DragHandle_default = DragHandle;

//#endregion
export { DragHandle_default as default };
//# sourceMappingURL=DragHandle.mjs.map