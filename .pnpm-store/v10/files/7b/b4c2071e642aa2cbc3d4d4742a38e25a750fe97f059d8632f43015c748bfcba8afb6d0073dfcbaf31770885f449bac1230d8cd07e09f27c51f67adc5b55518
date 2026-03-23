import Spine_default from "../../awesome/Spline/Spine.mjs";
import Loading_default from "./Loading.mjs";
import { memo, useState } from "react";
import { jsx, jsxs } from "react/jsx-runtime";
import { useThemeMode } from "antd-style";

//#region src/brand/LogoThree/LogoSpline.tsx
const LIGHT = "https://hub-apac-1.lobeobjects.space/light.splinecode";
const DARK = "https://hub-apac-1.lobeobjects.space/dark.splinecode";
const LogoSpline = memo(({ className, style, width, height, onLoad, ...rest }) => {
	const { isDarkMode } = useThemeMode();
	const [loading, setLoading] = useState(true);
	return /* @__PURE__ */ jsxs("div", {
		className,
		style: {
			height,
			position: "relative",
			width,
			...style
		},
		children: [loading && /* @__PURE__ */ jsx(Loading_default, {}), /* @__PURE__ */ jsx(Spine_default, {
			onLoad: (splineApp) => {
				setLoading(false);
				onLoad?.(splineApp);
			},
			scene: isDarkMode ? DARK : LIGHT,
			...rest
		})]
	});
});
var LogoSpline_default = LogoSpline;

//#endregion
export { LogoSpline_default as default };
//# sourceMappingURL=LogoSpline.mjs.map