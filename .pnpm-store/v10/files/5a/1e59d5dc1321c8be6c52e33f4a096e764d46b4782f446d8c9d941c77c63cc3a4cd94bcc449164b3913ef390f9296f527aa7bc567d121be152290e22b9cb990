import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Hotkey/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		borderless: css$1`
      ${staticStylish.variantBorderlessWithoutHover};
      padding-inline: 4px;
    `,
		filled: staticStylish.variantFilledWithoutHover,
		inverseThemeDark: css$1`
      color: ${cssVar$1.colorTextTertiary};
      background: color-mix(in srgb, ${cssVar$1.colorBgContainer} 8%, transparent);
    `,
		inverseThemeLight: css$1`
      color: ${cssVar$1.colorTextTertiary};
      background: color-mix(in srgb, ${cssVar$1.colorBgContainer} 16%, transparent);
    `,
		outlined: staticStylish.variantOutlinedWithoutHover,
		root: css$1`
      overflow: hidden;

      min-width: 1.8em;
      height: 1.8em;
      padding-block: 0;
      padding-inline: 8px;
      border: none;
      border-radius: ${cssVar$1.borderRadiusSM};

      font-family: ${cssVar$1.fontFamily};
      font-size: 12px;
      line-height: 1.1;
      color: ${cssVar$1.colorTextSecondary};
      text-align: center;
      white-space: nowrap;
    `
	};
});
const variants = cva(styles.root, {
	compoundVariants: [{
		class: styles.inverseThemeDark,
		inverseTheme: true,
		isDarkMode: true
	}, {
		class: styles.inverseThemeLight,
		inverseTheme: true,
		isDarkMode: false
	}],
	defaultVariants: {
		inverseTheme: false,
		isDarkMode: false,
		variant: "filled"
	},
	variants: {
		inverseTheme: {
			false: null,
			true: null
		},
		isDarkMode: {
			false: null,
			true: null
		},
		variant: {
			borderless: styles.borderless,
			filled: styles.filled,
			outlined: styles.outlined
		}
	}
});

//#endregion
export { variants };
//# sourceMappingURL=style.mjs.map