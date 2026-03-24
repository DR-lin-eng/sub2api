import { createStaticStyles, cx, keyframes } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Mermaid/SyntaxMermaid/style.ts
const fadeIn = keyframes`
  0% {
    opacity: 0;
  }
  100% {
    opacity: 1;
  }
`;
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		animated: css$1`
      img {
        opacity: 1;
        animation: ${fadeIn} 0.5s ease-in-out;
      }
    `,
		mermaid: cx("ant-mermaid-mermaid", css$1`
        img {
          display: block;
          width: 100%;
          height: auto;
        }
      `),
		noBackground: css$1`
      img {
        background: transparent !important;
      }
    `,
		noPadding: css$1`
      padding: 0;
    `,
		padding: css$1`
      padding: 16px;
    `,
		root: css$1`
      direction: ltr;
      margin: 0;
      padding: 0;
      text-align: start;
    `,
		unmermaid: css$1`
      color: ${cssVar$1.colorTextDescription};
    `
	};
});
const variants = cva(styles.root, {
	defaultVariants: {
		animated: false,
		mermaid: true,
		showBackground: false,
		variant: "borderless"
	},
	variants: {
		mermaid: {
			false: styles.unmermaid,
			true: styles.mermaid
		},
		showBackground: {
			false: styles.noBackground,
			true: null
		},
		animated: {
			true: styles.animated,
			false: null
		},
		variant: {
			filled: styles.padding,
			outlined: styles.padding,
			borderless: styles.noPadding
		}
	}
});

//#endregion
export { variants };
//# sourceMappingURL=style.mjs.map