import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, cx } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Input/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		borderless: css$1`
      &[class*='ant-input'] {
        ${staticStylish.variantBorderless}
        &:hover {
          ${staticStylish.variantBorderlessWithoutHover}
        }
      }
    `,
		borderlessOPT: css$1`
      &[class*='ant-otp'] {
        [class*='ant-otp-input'] {
          ${staticStylish.variantBorderless};
        }
      }
    `,
		filled: cx(staticStylish.variantFilled, css$1`
        &:focus-within {
          ${staticStylish.variantFilledWithoutHover}
        }
      `),
		filledOPT: css$1`
      &[class*='ant-otp'] {
        [class*='ant-otp-input'] {
          ${staticStylish.variantFilled};
        }
      }
    `,
		outlined: staticStylish.variantOutlined,
		outlinedOPT: css$1`
      &[class*='ant-otp'] {
        [class*='ant-otp-input'] {
          ${staticStylish.variantOutlined};
        }
      }
    `,
		root: css$1``,
		rootOPT: css$1`
      &[class*='ant-otp'] {
        [class*='ant-otp-input'] {
          &:focus-within {
            border-color: ${cssVar$1.colorBorder};
          }
        }
      }
    `,
		shadow: staticStylish.shadow,
		shadowOPT: css$1`
      &[class*='ant-otp'] {
        [class*='ant-otp-input'] {
          ${staticStylish.shadow};
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
const variantsOPT = cva(styles.rootOPT, {
	defaultVariants: { shadow: false },
	variants: {
		variant: {
			filled: styles.filledOPT,
			outlined: styles.outlinedOPT,
			borderless: styles.borderlessOPT,
			underlined: null
		},
		shadow: {
			false: null,
			true: styles.shadowOPT
		}
	}
});

//#endregion
export { variants, variantsOPT };
//# sourceMappingURL=style.mjs.map