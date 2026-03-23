import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Block/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		borderless: staticStylish.variantBorderlessWithoutHover,
		clickableBorderless: staticStylish.variantBorderless,
		clickableFilled: staticStylish.variantFilled,
		clickableOutlined: staticStylish.variantOutlined,
		clickableRoot: css$1`
      cursor: pointer;
    `,
		filled: staticStylish.variantFilledWithoutHover,
		glass: staticStylish.blur,
		outlined: staticStylish.variantOutlinedWithoutHover,
		root: css$1`
      position: relative;
      border-radius: ${cssVar$1.borderRadius};
    `,
		shadow: staticStylish.shadow
	};
});
const variants = cva(styles.root, {
	compoundVariants: [
		{
			class: styles.clickableBorderless,
			clickable: true,
			variant: "borderless"
		},
		{
			class: styles.clickableFilled,
			clickable: true,
			variant: "filled"
		},
		{
			class: styles.clickableOutlined,
			clickable: true,
			variant: "outlined"
		}
	],
	defaultVariants: {
		clickable: false,
		glass: false,
		shadow: false,
		variant: "filled"
	},
	variants: {
		variant: {
			filled: styles.filled,
			outlined: styles.outlined,
			borderless: styles.borderless
		},
		clickable: {
			false: null,
			true: styles.clickableRoot
		},
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