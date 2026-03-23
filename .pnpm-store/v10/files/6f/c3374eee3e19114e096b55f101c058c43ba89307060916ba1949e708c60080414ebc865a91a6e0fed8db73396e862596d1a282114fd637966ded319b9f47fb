import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, cx } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Menu/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => ({
	borderless: cx(staticStylish.variantBorderlessWithoutHover, css$1`
      padding: 0;
      border-radius: unset;
    `),
	compact: css$1`
    &[class*='ant-menu'] {
      [class*='ant-menu-item-divider'] {
        margin: 0;
      }
    }
  `,
	filled: staticStylish.variantFilledWithoutHover,
	outlined: staticStylish.variantOutlinedWithoutHover,
	root: css$1`
    &[class*='ant-menu'] {
      flex: 1;

      padding: 4px;
      border: none !important;
      border-radius: ${cssVar$1.borderRadiusLG};

      background: transparent;

      [class*='ant-menu-sub'][class*='ant-menu-inline'] {
        background: transparent;

        > [class*='ant-menu-item'] {
          padding-inline-start: 36px !important;
        }
      }

      [class*='ant-menu-item-divider'] {
        margin-block: 1em;
      }
    }
  `,
	shadow: staticStylish.shadow
}));
const variants = cva(styles.root, {
	defaultVariants: {
		compact: false,
		shadow: false,
		variant: "borderless"
	},
	variants: {
		variant: {
			filled: styles.filled,
			outlined: styles.outlined,
			borderless: styles.borderless
		},
		shadow: {
			false: null,
			true: styles.shadow
		},
		compact: {
			false: null,
			true: styles.compact
		}
	}
});

//#endregion
export { variants };
//# sourceMappingURL=style.mjs.map