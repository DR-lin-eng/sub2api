import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Avatar/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		borderless: staticStylish.variantBorderlessWithoutHover,
		filled: staticStylish.variantFilledWithoutHover,
		loading: css$1`
      position: absolute;
      color: #fff;
      background: ${cssVar$1.colorBgMask};
    `,
		outlined: staticStylish.variantOutlinedWithoutHover,
		root: css$1`
      flex: none;
      background: transparent;

      &[class*='ant-avatar'] {
        user-select: none;

        overflow: hidden;
        display: flex;
        align-items: center;
        justify-content: center;

        border: none;

        [class*='ant-avatar-string'] {
          transform: none !important;

          overflow: hidden;
          display: flex;
          align-items: center;
          justify-content: center;

          width: 100%;
          height: 100%;
          padding: 0;

          font-size: inherit;
          font-weight: bolder;
          line-height: 1;
          color: inherit;
        }
      }
    `,
		shadow: staticStylish.shadow
	};
});
const variants = cva(styles.root, {
	defaultVariants: {
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
		}
	}
});

//#endregion
export { styles, variants };
//# sourceMappingURL=style.mjs.map