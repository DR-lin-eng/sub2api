import { visit } from "../../node_modules/unist-util-visit/lib/index.mjs";

//#region src/Markdown/plugins/rehypeKatexDir.ts
const rehypeKatexDir = () => (tree) => {
	visit(tree, "element", (node) => {
		if (node.properties?.className?.includes("katex")) node.properties.dir = "ltr";
	});
};

//#endregion
export { rehypeKatexDir };
//# sourceMappingURL=rehypeKatexDir.mjs.map