import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/GuideCard/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => ({
	borderless: staticStylish.variantBorderlessWithoutHover,
	close: css$1`
    position: absolute;
    inset-block-start: 8px;
    inset-inline-end: 8px;
  `,
	content: css$1`
    padding: 16px;
  `,
	cover: css$1`
    align-self: center;
  `,
	desc: css$1`
    color: ${cssVar$1.colorTextDescription};
  `,
	filledDark: css$1`
    ${staticStylish.variantFilledWithoutHover};
    background: linear-gradient(
      to bottom,
      ${cssVar$1.colorFillTertiary},
      ${cssVar$1.colorFillQuaternary}
    );
  `,
	filledLight: css$1`
    ${staticStylish.variantFilledWithoutHover};
    background: linear-gradient(
      to bottom,
      ${cssVar$1.colorFillQuaternary},
      ${cssVar$1.colorFillTertiary}
    );
  `,
	outlined: staticStylish.variantOutlinedWithoutHover,
	root: css$1`
    position: relative;
    overflow: hidden;
    border-radius: ${cssVar$1.borderRadiusLG};
  `,
	shadow: staticStylish.shadow,
	title: css$1`
    font-size: 16px;
    font-weight: bold;
  `
}));
const variants = cva(styles.root, {
	compoundVariants: [{
		class: styles.filledDark,
		isDarkMode: true,
		variant: "filled"
	}, {
		class: styles.filledLight,
		isDarkMode: false,
		variant: "filled"
	}],
	defaultVariants: {
		isDarkMode: false,
		shadow: false,
		variant: "filled"
	},
	variants: {
		isDarkMode: {
			false: null,
			true: null
		},
		shadow: {
			false: null,
			true: styles.shadow
		},
		variant: {
			borderless: styles.borderless,
			filled: null,
			outlined: styles.outlined
		}
	}
});

//#endregion
export { styles, variants };
//# sourceMappingURL=style.mjs.map