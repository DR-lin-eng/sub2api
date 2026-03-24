'use client';

import Center_default from "../Flex/Center.mjs";
import { styles } from "./style.mjs";
import FileIcon_default from "./components/FileIcon.mjs";
import FolderIcon_default from "./components/FolderIcon.mjs";
import { useMemo } from "react";
import { jsx, jsxs } from "react/jsx-runtime";
import { cssVar, cx, useThemeMode } from "antd-style";

//#region src/FileTypeIcon/FileTypeIcon.tsx
const FileTypeIcon = ({ icon, color, filetype, type = "file", size = 48, style, variant, className, ref, ...rest }) => {
	const { isDarkMode } = useThemeMode();
	const isMono = variant === "mono";
	const filetypeShort = useMemo(() => filetype && filetype.length > 4 ? filetype.slice(0, 4) : filetype, [filetype]);
	const fontSize = useMemo(() => {
		if (filetypeShort && filetypeShort.length > 3) return 24 / (4 + (filetypeShort.length - 3));
		return 6;
	}, [filetypeShort]);
	const iconColor = useMemo(() => isMono ? isDarkMode ? cssVar.colorFill : cssVar.colorBgContainer : color || cssVar.geekblue, [
		isMono,
		isDarkMode,
		color
	]);
	const content = type === "file" ? /* @__PURE__ */ jsx(FileIcon_default, {
		filetypeShort,
		fontSize,
		hasIcon: !!icon,
		iconColor,
		isMono,
		size,
		...rest
	}) : /* @__PURE__ */ jsx(FolderIcon_default, {
		filetype,
		fontSize,
		hasIcon: !!icon,
		iconColor,
		isMono,
		size,
		...rest
	});
	if (!icon) return content;
	return /* @__PURE__ */ jsxs(Center_default, {
		className: cx(styles.container, className),
		flex: "none",
		height: size,
		ref,
		style,
		width: size,
		...rest,
		children: [/* @__PURE__ */ jsx("div", {
			className: styles.inner,
			style: {
				fontSize: size / 2.4,
				top: type === "file" ? "20%" : "16%"
			},
			children: icon
		}), content]
	});
};
FileTypeIcon.displayName = "FileTypeIcon";
var FileTypeIcon_default = FileTypeIcon;

//#endregion
export { FileTypeIcon_default as default };
//# sourceMappingURL=FileTypeIcon.mjs.map