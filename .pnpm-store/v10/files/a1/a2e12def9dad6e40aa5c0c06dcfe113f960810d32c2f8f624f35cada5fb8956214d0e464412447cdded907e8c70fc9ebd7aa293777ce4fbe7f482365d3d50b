import { createStaticStyles, keyframes } from "antd-style";

//#region src/Markdown/SyntaxMarkdown/style.ts
const fadeIn = keyframes`
    0% {
      opacity: 0;
    }
    100% {
      opacity: 1;
    }
  `;
const styles = createStaticStyles(({ css: css$1 }) => {
	return { animated: css$1`
      .animate-fade-in,
      .katex-html span,
      span.line > span,
      code:not(:has(span.line)) {
        opacity: 1;
        animation: ${fadeIn} 1s ease-in-out;
      }
    ` };
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map