import geekblue_default from "../../../color/colors/geekblue.mjs";
import gold_default from "../../../color/colors/gold.mjs";
import gray_default from "../../../color/colors/gray.mjs";
import green_default from "../../../color/colors/green.mjs";
import primary_default from "../../../color/colors/primary.mjs";
import volcano_default from "../../../color/colors/volcano.mjs";
import { generateColorNeutralPalette, generateColorPalette } from "../generateColorPalette.mjs";

//#region src/styles/theme/token/light.ts
const primaryToken = generateColorPalette({
	appearance: "light",
	scale: primary_default,
	type: "Primary"
});
const neutralToken = generateColorNeutralPalette({
	appearance: "light",
	scale: gray_default
});
const successToken = generateColorPalette({
	appearance: "light",
	scale: green_default,
	type: "Success"
});
const warningToken = generateColorPalette({
	appearance: "light",
	scale: gold_default,
	type: "Warning"
});
const errorToken = generateColorPalette({
	appearance: "light",
	scale: volcano_default,
	type: "Error"
});
const infoToken = generateColorPalette({
	appearance: "light",
	scale: geekblue_default,
	type: "Info"
});
const lightBaseToken = {
	...primaryToken,
	...neutralToken,
	...successToken,
	...warningToken,
	...errorToken,
	...infoToken,
	boxShadow: "0 20px 20px -8px rgba(0, 0, 0, 0.24)",
	boxShadowSecondary: "0 8px 16px -4px rgba(0, 0, 0, 0.2)",
	boxShadowTertiary: "0 3px 1px -1px rgba(26, 26, 26, 0.06)",
	colorLink: infoToken.colorInfoText,
	colorLinkActive: infoToken.colorInfoTextActive,
	colorLinkHover: infoToken.colorInfoTextHover,
	colorTextLightSolid: neutralToken.colorBgLayout
};
var light_default = lightBaseToken;

//#endregion
export { light_default as default };
//# sourceMappingURL=light.mjs.map