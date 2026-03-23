import { visit } from "../../node_modules/unist-util-visit/lib/index.mjs";

//#region src/Markdown/plugins/rehypeStreamAnimated.ts
const rehypeStreamAnimated = () => {
	return (tree) => {
		visit(tree, "element", ((node) => {
			if ([
				"p",
				"h1",
				"h2",
				"h3",
				"h4",
				"h5",
				"h6",
				"li",
				"strong"
			].includes(node.tagName) && node.children) {
				const newChildren = [];
				for (const child of node.children) if (child.type === "text") [...new Intl.Segmenter("zh", { granularity: "word" }).segment(child.value)].map((segment) => segment.segment).filter(Boolean).forEach((word) => {
					newChildren.push({
						children: [{
							type: "text",
							value: word
						}],
						properties: { className: "animate-fade-in" },
						tagName: "span",
						type: "element"
					});
				});
				else newChildren.push(child);
				node.children = newChildren;
			}
		}));
	};
};

//#endregion
export { rehypeStreamAnimated };
//# sourceMappingURL=rehypeStreamAnimated.mjs.map