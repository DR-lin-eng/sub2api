'use client';

import { useCdnFn } from "../../ConfigProvider/index.mjs";
import FlexBasic_default from "../../Flex/FlexBasic.mjs";
import Img_default from "../../Img/index.mjs";
import Spine_default from "../../awesome/Spline/Spine.mjs";
import { memo, useState } from "react";
import { jsx, jsxs } from "react/jsx-runtime";

//#region src/brand/LogoThree/index.tsx
const LOGO_3D = {
	path: "assets/logo-3d.webp",
	pkg: "@lobehub/assets-logo",
	version: "1.2.0"
};
const LogoThree = memo(({ className, style, size = 128, onLoad, ...rest }) => {
	const genCdnUrl = useCdnFn();
	const [loading, setLoading] = useState(true);
	return /* @__PURE__ */ jsxs(FlexBasic_default, {
		align: "center",
		className,
		flex: "none",
		justify: "center",
		style: {
			height: size,
			overflow: "hidden",
			position: "relative",
			width: size,
			...style
		},
		children: [loading && /* @__PURE__ */ jsx(Img_default, {
			alt: "logo",
			height: size * .75,
			src: genCdnUrl(LOGO_3D),
			style: { position: "absolute" },
			width: size * .75
		}), /* @__PURE__ */ jsx(Spine_default, {
			onLoad: (splineApp) => {
				setLoading(false);
				onLoad?.(splineApp);
			},
			scene: "https://hub-apac-1.lobeobjects.space/logo.splinecode",
			style: {
				flex: "none",
				height: size,
				width: size
			},
			...rest
		})]
	});
});
LogoThree.displayName = "LobeHubLogoThree";
var LogoThree_default = LogoThree;

//#endregion
export { LogoThree_default as default };
//# sourceMappingURL=index.mjs.map