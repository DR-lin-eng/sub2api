import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Segmented/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		borderless: staticStylish.variantBorderlessWithoutHover,
		filled: css$1`
      border: 1px solid ${cssVar$1.colorFillQuaternary};
      background: ${cssVar$1.colorBgLayout};
    `,
		glass: staticStylish.blur,
		outlined: css$1`
      border: 1px solid ${cssVar$1.colorBorderSecondary};
      background: transparent;
    `,
		root: css$1``,
		shadow: staticStylish.shadow
	};
});
const variants = cva(styles.root, {
	defaultVariants: {
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