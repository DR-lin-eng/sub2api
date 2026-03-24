import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/SortableList/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		borderless: staticStylish.variantBorderlessWithoutHover,
		container: css$1`
      padding: 0;
      list-style: none;
    `,
		filled: staticStylish.variantFilledWithoutHover,
		item: css$1`
      overflow: hidden;
      box-sizing: border-box;
      border-radius: ${cssVar$1.borderRadius};
      list-style: none;
    `,
		itemVariant: css$1`
      padding-block: 4px;
      padding-inline: 4px 16px;
    `,
		outlined: staticStylish.variantOutlinedWithoutHover
	};
});
const variants = cva(styles.item, {
	compoundVariants: [{
		className: styles.itemVariant,
		variant: "outlined"
	}, {
		className: styles.itemVariant,
		variant: "filled"
	}],
	defaultVariants: { variant: "borderless" },
	variants: { variant: {
		filled: styles.filled,
		outlined: styles.outlined,
		borderless: styles.borderless
	} }
});

//#endregion
export { styles, variants };
//# sourceMappingURL=style.mjs.map