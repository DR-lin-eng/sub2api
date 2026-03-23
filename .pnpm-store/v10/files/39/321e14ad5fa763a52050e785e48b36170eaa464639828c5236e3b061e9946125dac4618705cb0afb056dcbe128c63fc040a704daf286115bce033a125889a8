import { memo, useCallback } from "react";
import { Fragment as Fragment$1, jsx, jsxs } from "react/jsx-runtime";
import { isUndefined } from "es-toolkit/compat";

//#region src/awesome/GridBackground/components/Grid.tsx
var Line = /* @__PURE__ */ function(Line$1) {
	Line$1[Line$1["l7"] = 0] = "l7";
	Line$1[Line$1["l6"] = 1] = "l6";
	Line$1[Line$1["l5"] = 2] = "l5";
	Line$1[Line$1["l4"] = 3] = "l4";
	Line$1[Line$1["l3"] = 4] = "l3";
	Line$1[Line$1["l2"] = 5] = "l2";
	Line$1[Line$1["l1"] = 6] = "l1";
	Line$1[Line$1["center"] = 7] = "center";
	Line$1[Line$1["r1"] = 8] = "r1";
	Line$1[Line$1["r2"] = 9] = "r2";
	Line$1[Line$1["r3"] = 10] = "r3";
	Line$1[Line$1["r4"] = 11] = "r4";
	Line$1[Line$1["r5"] = 12] = "r5";
	Line$1[Line$1["r6"] = 13] = "r6";
	Line$1[Line$1["r7"] = 14] = "r7";
	return Line$1;
}(Line || {});
const Grid = memo(({ color = "#fff", strokeWidth = 3, linePick, ...rest }) => {
	const isUnpick = isUndefined(linePick);
	const showLine = useCallback((l) => isUnpick || linePick === l, [linePick]);
	const vLine = /* @__PURE__ */ jsxs(Fragment$1, { children: [
		showLine(Line.l7) && /* @__PURE__ */ jsx("path", { d: "M2 420v-60.343c0-21.82 14.15-41.12 34.959-47.684L1026 0h0" }),
		showLine(Line.l6) && /* @__PURE__ */ jsx("path", { d: "M268 420v-62.077c0-20.977 13.094-39.724 32.789-46.944L1149 0h0" }),
		showLine(Line.l5) && /* @__PURE__ */ jsx("path", { d: "M534 420v-64.358a50 50 0 0129.884-45.775L1269 0h0" }),
		showLine(Line.l4) && /* @__PURE__ */ jsx("path", { d: "M800 420v-67.395a50 50 0 0125.958-43.84L1389 0h0" }),
		showLine(Line.l3) && /* @__PURE__ */ jsx("path", { d: "M1066 420v-71.645a50 50 0 0120.456-40.337L1507 0h0" }),
		showLine(Line.l2) && /* @__PURE__ */ jsx("path", { d: "M1332 420v-77.506a50 50 0 0113.194-33.843L1629 0h0" }),
		showLine(Line.l1) && /* @__PURE__ */ jsx("path", { d: "M1598 420v-86.225a50 50 0 014.438-20.594L1744 0h0" }),
		showLine(Line.center) && /* @__PURE__ */ jsx("path", { d: "M1864 420V0h0" }),
		showLine(Line.r1) && /* @__PURE__ */ jsx("path", { d: "M2130 420v-86.225a50 50 0 00-4.438-20.594L1984 0h0" }),
		showLine(Line.r2) && /* @__PURE__ */ jsx("path", { d: "M2396 420v-77.506a50 50 0 00-13.194-33.843L2099 0h0" }),
		showLine(Line.r3) && /* @__PURE__ */ jsx("path", { d: "M2662 420v-71.645a50 50 0 00-20.456-40.337L2221 0h0" }),
		showLine(Line.r4) && /* @__PURE__ */ jsx("path", { d: "M2928 420v-67.395a50 50 0 00-25.958-43.84L2339 0h0" }),
		showLine(Line.r5) && /* @__PURE__ */ jsx("path", { d: "M3194 420v-64.358a50 50 0 00-29.884-45.775L2459 0h0" }),
		showLine(Line.r6) && /* @__PURE__ */ jsx("path", { d: "M3460 420v-62.077c0-20.977-13.094-39.724-32.789-46.944L2579 0h0" }),
		showLine(Line.r7) && /* @__PURE__ */ jsx("path", { d: "M3726 420v-60.343c0-21.82-14.15-41.12-34.959-47.684L2702 0h0" })
	] });
	const hLine = isUnpick && /* @__PURE__ */ jsxs(Fragment$1, { children: [
		/* @__PURE__ */ jsx("path", { d: "M2835 42H892" }),
		/* @__PURE__ */ jsx("path", { d: "M595 136h2538" }),
		/* @__PURE__ */ jsx("path", { d: "M237 249h3254" })
	] });
	return /* @__PURE__ */ jsx("svg", {
		style: { width: "100%" },
		viewBox: "0 0 3728 422",
		xmlns: "http://www.w3.org/2000/svg",
		...rest,
		children: /* @__PURE__ */ jsxs("g", {
			fill: "none",
			fillRule: "evenodd",
			stroke: color,
			strokeWidth,
			children: [vLine, hLine]
		})
	});
});
Grid.displayName = "Grid";
var Grid_default = Grid;

//#endregion
export { Grid_default as default };
//# sourceMappingURL=Grid.mjs.map