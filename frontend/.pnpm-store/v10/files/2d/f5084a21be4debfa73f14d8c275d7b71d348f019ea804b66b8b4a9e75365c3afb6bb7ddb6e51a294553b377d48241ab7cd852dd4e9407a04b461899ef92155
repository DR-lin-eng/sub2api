import blue_default from "../../../color/colors/blue.mjs";
import gold_default from "../../../color/colors/gold.mjs";
import gray_default from "../../../color/colors/gray.mjs";
import lime_default from "../../../color/colors/lime.mjs";
import primary_default from "../../../color/colors/primary.mjs";
import red_default from "../../../color/colors/red.mjs";
import { generateColorNeutralPalette, generateColorPalette } from "../generateColorPalette.mjs";

//#region src/styles/theme/token/dark.ts
const primaryToken = generateColorPalette({
	appearance: "dark",
	scale: primary_default,
	type: "Primary"
});
const neutralToken = generateColorNeutralPalette({
	appearance: "dark",
	scale: gray_default
});
const successToken = generateColorPalette({
	appearance: "dark",
	scale: lime_default,
	type: "Success"
});
const warningToken = generateColorPalette({
	appearance: "dark",
	scale: gold_default,
	type: "Warning"
});
const errorToken = generateColorPalette({
	appearance: "dark",
	scale: red_default,
	type: "Error"
});
const infoToken = generateColorPalette({
	appearance: "dark",
	scale: blue_default,
	type: "Info"
});
const darkBaseToken = {
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
var dark_default = darkBaseToken;

//#endregion
export { dark_default as default };
//# sourceMappingURL=dark.mjs.map