import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, cx } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Highlighter/style.ts
const actionsHoverCls = "ant-highlighter-highlighter-hover-actions";
const langHoverCls = "ant-highlighter-highlighter-hover-lang";
const expandCls = "ant-highlighter-highlighter-body-expand";
const prefix = "ant-highlighter";
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		actions: cx(actionsHoverCls, css$1`
        position: absolute;
        z-index: 2;
        inset-block-start: 8px;
        inset-inline-end: 8px;

        opacity: 0;
      `),
		bodyCollapsed: css$1`
      height: 0;
      opacity: 0;
    `,
		bodyExpand: cx(expandCls),
		bodyRoot: css$1`
      overflow: hidden;
      transition: opacity 0.25s ${cssVar$1.motionEaseOut};
    `,
		borderless: staticStylish.variantBorderlessWithoutHover,
		filled: cx(staticStylish.variantFilledWithoutHover, css$1`
        background: ${cssVar$1.colorFillQuaternary};
      `),
		headerBorderless: css$1`
      padding-inline: 0;
    `,
		headerFilled: css$1`
      background: transparent;
    `,
		headerOutlined: css$1`
      & + .${expandCls} {
        border-block-start: 1px solid ${cssVar$1.colorFillQuaternary};
      }
    `,
		headerRoot: css$1`
      cursor: pointer;
      position: relative;
      padding: 4px;
    `,
		lang: cx(langHoverCls, staticStylish.blur, css$1`
        position: absolute;
        z-index: 2;
        inset-block-end: 8px;
        inset-inline-end: 8px;

        font-family: ${cssVar$1.fontFamilyCode};
        color: ${cssVar$1.colorTextSecondary};

        opacity: 0;
        background: ${cssVar$1.colorFillQuaternary};

        transition: opacity 0.1s;
      `),
		nowrap: css$1`
      pre,
      code {
        text-wrap: nowrap;
      }
    `,
		outlined: staticStylish.variantOutlinedWithoutHover,
		root: cx(prefix, css$1`
        position: relative;

        overflow: hidden;

        width: 100%;
        border-radius: ${cssVar$1.borderRadius};

        transition: background-color 100ms ${cssVar$1.motionEaseOut};

        .languageTitle {
          opacity: 0.5;
          filter: grayscale(100%);
          transition:
            opacity,
            grayscale 0.2s ${cssVar$1.motionEaseInOut};
        }

        .panel-actions {
          opacity: 0;
          transition: opacity 0.2s ${cssVar$1.motionEaseInOut};
        }

        &:hover {
          .languageTitle {
            opacity: 1;
            filter: grayscale(0%);
          }

          .panel-actions {
            opacity: 1;
          }

          .${actionsHoverCls} {
            opacity: 1;
          }

          .${langHoverCls} {
            opacity: 1;
          }
        }

        pre {
          height: 100%;
          font-size: 12px;
        }

        code {
          background: transparent !important;
        }
      `),
		shadow: staticStylish.shadow,
		wrap: css$1`
      pre,
      code {
        text-wrap: wrap;
      }
    `
	};
});
const variants = cva(styles.root, {
	defaultVariants: {
		shadow: false,
		variant: "filled",
		wrap: false
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
		wrap: {
			false: styles.nowrap,
			true: styles.wrap
		}
	}
});
const headerVariants = cva(styles.headerRoot, {
	defaultVariants: { variant: "filled" },
	variants: { variant: {
		filled: cx(styles.headerFilled, styles.headerOutlined),
		outlined: styles.headerOutlined,
		borderless: styles.headerBorderless
	} }
});
const bodyVariants = cva(styles.bodyRoot, {
	defaultVariants: { expand: true },
	variants: { expand: {
		false: styles.bodyCollapsed,
		true: styles.bodyExpand
	} }
});

//#endregion
export { bodyVariants, headerVariants, styles, variants };
//# sourceMappingURL=style.mjs.map