import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Tag/styles.ts
const prefixCls = "ant";
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => ({
	borderless: staticStylish.variantBorderlessWithoutHover,
	filled: staticStylish.variantFilledWithoutHover,
	large: css$1`
    &.${prefixCls}-tag {
      height: 28px;
      padding-inline: 12px;
      border-radius: 6px !important;
    }
  `,
	outlined: staticStylish.variantOutlinedWithoutHover,
	root: css$1`
    color: ${cssVar$1.colorTextSecondary};

    &.${prefixCls}-tag {
      user-select: none;

      display: flex;
      gap: 0.4em;
      align-items: center;
      justify-content: center;

      width: fit-content;
      height: 22px;
      margin: 0;
      border-radius: 3px;

      line-height: 1.2;

      span {
        margin: 0;
      }

      span:not(.anticon) {
        line-height: inherit;
      }
    }
  `,
	small: css$1`
    &.${prefixCls}-tag {
      height: 20px;
      padding-inline: 4px;
      border-radius: 3px;
    }
  `
}));
const variants = cva(styles.root, {
	defaultVariants: {
		size: "middle",
		variant: "filled"
	},
	variants: {
		variant: {
			filled: styles.filled,
			outlined: styles.outlined,
			borderless: styles.borderless
		},
		size: {
			small: styles.small,
			middle: null,
			large: styles.large
		}
	}
});

//#endregion
export { variants };
//# sourceMappingURL=styles.mjs.map