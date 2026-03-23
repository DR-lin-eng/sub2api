import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Button/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		glass: staticStylish.blur,
		root: css$1`
      &[class*='ant-btn'] {
        > [class*='ant-btn-icon'] {
          display: flex;
        }
      }
    `,
		shadow: css$1`
      box-shadow: ${cssVar$1.boxShadowTertiary} !important;
    `
	};
});
const variants = cva(styles.root, {
	defaultVariants: {
		glass: false,
		shadow: false
	},
	variants: {
		glass: {
			false: null,
			true: styles.glass
		},
		shadow: {
			false: null,
			true: styles.shadow
		}
	}
});

//#endregion
export { variants };
//# sourceMappingURL=style.mjs.map