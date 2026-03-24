import blue_default from "../color/colors/blue.mjs";
import cyan_default from "../color/colors/cyan.mjs";
import geekblue_default from "../color/colors/geekblue.mjs";
import gold_default from "../color/colors/gold.mjs";
import green_default from "../color/colors/green.mjs";
import lime_default from "../color/colors/lime.mjs";
import magenta_default from "../color/colors/magenta.mjs";
import orange_default from "../color/colors/orange.mjs";
import purple_default from "../color/colors/purple.mjs";
import red_default from "../color/colors/red.mjs";
import volcano_default from "../color/colors/volcano.mjs";
import yellow_default from "../color/colors/yellow.mjs";
import mauve_default from "../color/neutrals/mauve.mjs";
import olive_default from "../color/neutrals/olive.mjs";
import sage_default from "../color/neutrals/sage.mjs";
import sand_default from "../color/neutrals/sand.mjs";
import slate_default from "../color/neutrals/slate.mjs";

//#region src/styles/customTheme.ts
const primaryColors = {
	blue: blue_default.dark[9],
	cyan: cyan_default.dark[9],
	geekblue: geekblue_default.dark[9],
	gold: gold_default.dark[9],
	green: green_default.dark[9],
	lime: lime_default.dark[9],
	magenta: magenta_default.dark[9],
	orange: orange_default.dark[9],
	purple: purple_default.dark[9],
	red: red_default.dark[9],
	volcano: volcano_default.dark[9],
	yellow: yellow_default.dark[9]
};
const primaryColorsSwatches = [
	primaryColors.red,
	primaryColors.orange,
	primaryColors.gold,
	primaryColors.yellow,
	primaryColors.lime,
	primaryColors.green,
	primaryColors.cyan,
	primaryColors.blue,
	primaryColors.geekblue,
	primaryColors.purple,
	primaryColors.magenta,
	primaryColors.volcano
];
const neutralColors = {
	mauve: mauve_default.dark[9],
	olive: olive_default.dark[9],
	sage: sage_default.dark[9],
	sand: sand_default.dark[9],
	slate: slate_default.dark[9]
};
const neutralColorsSwatches = [
	neutralColors.mauve,
	neutralColors.slate,
	neutralColors.sage,
	neutralColors.olive,
	neutralColors.sand
];
const findCustomThemeName = (type, value) => {
	const res = type === "primary" ? primaryColors : neutralColors;
	return Object.entries(res).find((item) => item[1] === value)?.[0];
};

//#endregion
export { findCustomThemeName, neutralColors, neutralColorsSwatches, primaryColors, primaryColorsSwatches };
//# sourceMappingURL=customTheme.mjs.map