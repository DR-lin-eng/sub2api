import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/HotkeyInput/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		borderless: staticStylish.variantBorderless,
		disabled: staticStylish.disabled,
		error: css$1`
      border: 1px solid ${cssVar$1.colorError};
    `,
		errorText: css$1`
      font-size: 12px;
      color: ${cssVar$1.colorError};
    `,
		filled: staticStylish.variantFilled,
		focused: css$1`
      background: ${cssVar$1.colorFillSecondary} !important;
    `,
		hiddenInput: css$1`
      cursor: text;

      position: absolute;
      z-index: -1;
      inset-block-start: 0;
      inset-inline-start: 0;

      width: 100%;
      height: 100%;

      opacity: 0;
    `,
		outlined: staticStylish.variantOutlined,
		placeholder: css$1`
      color: ${cssVar$1.colorTextDescription};
    `,
		root: css$1`
      cursor: pointer;

      position: relative;

      max-width: 100%;
      height: 36px;
      padding-block: 0;
      padding-inline: 12px;
      border-radius: ${cssVar$1.borderRadius};
    `,
		shadow: staticStylish.shadow
	};
});
const variants = cva(styles.root, {
	defaultVariants: {
		disabled: false,
		error: false,
		shadow: false,
		variant: "outlined"
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
		focused: {
			false: null,
			true: styles.focused
		},
		error: {
			fales: null,
			true: styles.error
		},
		disabled: {
			false: null,
			true: styles.disabled
		}
	}
});

//#endregion
export { styles, variants };
//# sourceMappingURL=style.mjs.map