import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/ActionIcon/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		active: staticStylish.active,
		borderless: staticStylish.variantBorderless,
		dangerBorderless: staticStylish.variantBorderlessDanger,
		dangerFilled: staticStylish.variantFilledDanger,
		dangerOutlined: staticStylish.variantOutlinedDanger,
		dangerRoot: css$1`
      &:hover {
        color: ${cssVar$1.colorError};
      }

      &:active {
        color: ${cssVar$1.colorErrorActive};
      }
    `,
		disabled: staticStylish.disabled,
		filled: staticStylish.variantFilled,
		glass: staticStylish.blur,
		outlined: staticStylish.variantOutlined,
		root: css$1`
      cursor: pointer;

      position: relative;

      overflow: hidden;

      color: ${cssVar$1.colorTextTertiary};

      transition:
        color 400ms ${cssVar$1.motionEaseOut},
        background 100ms ${cssVar$1.motionEaseOut};

      &:hover {
        color: ${cssVar$1.colorTextSecondary};
      }

      &:active {
        color: ${cssVar$1.colorText};
      }
    `,
		shadow: staticStylish.shadow
	};
});
const variants = cva(styles.root, {
	compoundVariants: [
		{
			className: styles.dangerFilled,
			danger: true,
			variant: "filled"
		},
		{
			className: styles.dangerBorderless,
			danger: true,
			variant: "borderless"
		},
		{
			className: styles.dangerOutlined,
			danger: true,
			variant: "outlined"
		}
	],
	defaultVariants: {
		active: false,
		danger: false,
		disabled: false,
		glass: false,
		shadow: false,
		variant: "borderless"
	},
	variants: {
		variant: {
			filled: styles.filled,
			outlined: styles.outlined,
			borderless: styles.borderless
		},
		glass: {
			false: null,
			true: styles.glass
		},
		shadow: {
			false: null,
			true: styles.shadow
		},
		active: {
			false: null,
			true: styles.active
		},
		danger: {
			false: null,
			true: styles.dangerRoot
		},
		disabled: {
			false: null,
			true: styles.disabled
		}
	}
});

//#endregion
export { variants };
//# sourceMappingURL=style.mjs.map