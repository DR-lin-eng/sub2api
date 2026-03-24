import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/AutoComplete/style.ts
const styles = createStaticStyles(({ css: css$1 }) => {
	return {
		borderless: css$1`
      &[class*='ant-select'] {
        > [class*='ant-select-selector'] {
          ${staticStylish.variantBorderless}
        }
      }
    `,
		filled: css$1`
      &[class*='ant-select'] {
        > [class*='ant-select-selector'] {
          ${staticStylish.variantFilled}
        }
      }
    `,
		outlined: css$1`
      &[class*='ant-select'] {
        > [class*='ant-select-selector'] {
          ${staticStylish.variantOutlined}
        }
      }
    `,
		root: css$1``,
		shadow: css$1`
      &[class*='ant-select'] {
        > [class*='ant-select-selector'] {
          ${staticStylish.shadow}
        }
      }
    `
	};
});
const variants = cva(styles.root, {
	defaultVariants: { shadow: false },
	variants: {
		variant: {
			filled: styles.filled,
			outlined: styles.outlined,
			borderless: styles.borderless,
			underlined: null
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