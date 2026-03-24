'use client';

import { useCdnFn } from "../ConfigProvider/index.mjs";
import Center_default from "../Flex/Center.mjs";
import Img_default from "../Img/index.mjs";
import FileTypeIcon_default from "../FileTypeIcon/FileTypeIcon.mjs";
import { getIconUrlForDirectoryPath, getIconUrlForFilePath } from "./utils.mjs";
import { useMemo } from "react";
import { jsx, jsxs } from "react/jsx-runtime";

//#region src/MaterialFileTypeIcon/MaterialFileTypeIcon.tsx
const MaterialFileTypeIcon = ({ fallbackUnknownType = true, filename, size = 48, variant = "raw", type, style, open, ...rest }) => {
	const ICONS_URL = useCdnFn()({
		path: "assets",
		pkg: "@lobehub/assets-fileicon",
		version: "1.0.0"
	});
	const iconUrl = useMemo(() => {
		return type === "file" ? getIconUrlForFilePath({
			fallbackUnknownType,
			iconsUrl: ICONS_URL,
			path: filename
		}) : getIconUrlForDirectoryPath({
			fallbackUnknownType,
			iconsUrl: ICONS_URL,
			open,
			path: filename
		});
	}, [
		ICONS_URL,
		type,
		filename,
		open
	]);
	if (!iconUrl) return /* @__PURE__ */ jsx(FileTypeIcon_default, {
		filetype: filename.split(".")[1],
		size,
		type,
		variant: "mono"
	});
	if (variant === "raw") return /* @__PURE__ */ jsx(Img_default, {
		alt: filename,
		height: size,
		src: iconUrl,
		style,
		width: size,
		...rest
	});
	return /* @__PURE__ */ jsxs(Center_default, {
		flex: "none",
		height: size,
		style: {
			position: "relative",
			...style
		},
		width: size,
		...rest,
		children: [/* @__PURE__ */ jsx(FileTypeIcon_default, {
			size,
			type: variant,
			variant: "mono"
		}), /* @__PURE__ */ jsx(Img_default, {
			alt: filename,
			height: size / 2,
			src: iconUrl,
			style: {
				position: "absolute",
				top: size / 3
			},
			width: size / 2,
			...rest
		})]
	});
};
MaterialFileTypeIcon.displayName = "MaterialFileTypeIcon";
var MaterialFileTypeIcon_default = MaterialFileTypeIcon;

//#endregion
export { MaterialFileTypeIcon_default as default };
//# sourceMappingURL=MaterialFileTypeIcon.mjs.map