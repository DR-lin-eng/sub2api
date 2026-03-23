import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, cx } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/CodeEditor/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		borderless: cx(staticStylish.variantBorderlessWithoutHover, css$1`
        border-radius: 0;

        pre,
        textarea {
          padding: 0;
        }
      `),
		filled: staticStylish.variantFilledWithoutHover,
		highlight: css$1`
      pointer-events: none;
    `,
		outlined: staticStylish.variantOutlinedWithoutHover,
		root: css$1`
      position: relative;

      overflow: hidden auto;

      width: 100%;
      height: fit-content;
      border-radius: ${cssVar$1.borderRadius};

      font-size: 12px;

      pre,
      textarea {
        margin: 0;
        padding: 16px;
      }

      textarea,
      pre,
      code {
        overflow: hidden;

        font-family: ${cssVar$1.fontFamilyCode};
        font-size: inherit;
        line-height: inherit;
        word-break: inherit;
        word-wrap: break-word;
        white-space: pre-wrap;
      }
    `,
		textarea: css$1`
      resize: none;

      position: absolute;
      inset-block-start: 0;
      inset-inline-start: 0;

      overflow: hidden;

      box-sizing: border-box;
      width: 100%;
      height: 100%;
      padding: 0;
      border: none;

      color: transparent;
      text-align: start;

      background: transparent;
      outline: none;
      caret-color: ${cssVar$1.colorText};

      &::placeholder {
        color: ${cssVar$1.colorTextQuaternary};
      }

      &:focus {
        border: none;
        outline: none;
        box-shadow: none;
      }
    `
	};
});
const variants = cva(styles.root, {
	defaultVariants: { variant: "borderless" },
	variants: { variant: {
		filled: styles.filled,
		outlined: styles.outlined,
		borderless: styles.borderless
	} }
});

//#endregion
export { styles, variants };
//# sourceMappingURL=style.mjs.map