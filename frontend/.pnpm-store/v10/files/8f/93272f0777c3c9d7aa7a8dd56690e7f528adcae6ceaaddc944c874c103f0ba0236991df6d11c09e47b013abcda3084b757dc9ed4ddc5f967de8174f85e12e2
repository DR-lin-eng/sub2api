import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Collapse/style.ts
const DEFAULT_PADDING = "12px 16px";
const getPadding = (padding) => !padding && padding !== 0 ? DEFAULT_PADDING : `${typeof padding === "string" ? padding : `${padding}px`} !important`;
const prefixCls = "ant";
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		borderless: css$1`
      &.${prefixCls}-collapse {
        .${prefixCls}-collapse-header {
          padding-inline: 0;
        }
        .${prefixCls}-collapse-panel {
          padding-inline: 0;
          .${prefixCls}-collapse-body {
            padding-inline: 0;
          }
        }
      }
    `,
		desc: css$1`
      font-size: 12px;
      color: ${cssVar$1.colorTextDescription};
    `,
		filledDark: css$1`
      &.${prefixCls}-collapse {
        .${prefixCls}-collapse-item {
          background: ${cssVar$1.colorBgLayout};
          .${prefixCls}-collapse-panel {
            margin-inline: 3px;
            margin-block-end: 3px;
            border-radius: ${cssVar$1.borderRadius};
            ${staticStylish.variantOutlinedWithoutHover};
          }
        }
      }
    `,
		filledLight: css$1`
      &.${prefixCls}-collapse {
        .${prefixCls}-collapse-item {
          background: ${cssVar$1.colorFillQuaternary};
          .${prefixCls}-collapse-panel {
            margin-inline: 3px;
            margin-block-end: 3px;
            border-radius: ${cssVar$1.borderRadius};
            ${staticStylish.variantOutlinedWithoutHover};
            background: ${cssVar$1.colorBgContainer};
            ${staticStylish.shadow};
          }
        }
      }
    `,
		gapOutlined: css$1`
      &.${prefixCls}-collapse {
        border: none;
        background: transparent;
        .${prefixCls}-collapse-item {
          border: 1px solid ${cssVar$1.colorFillSecondary};
          background: ${cssVar$1.colorBgContainer};
        }

        .${prefixCls}-collapse-item:not(:first-child) {
          .${prefixCls}-collapse-header {
            border-block-start: none;
          }
        }
      }
    `,
		gapRoot: css$1`
      &.${prefixCls}-collapse {
        display: flex;
        flex-direction: column;
        border: none;
        box-shadow: none;
        .${prefixCls}-collapse-item {
          border: none;
          border-radius: ${cssVar$1.borderRadiusLG};
        }
      }
    `,
		hideCollapsibleIcon: css$1`
      .${prefixCls}-collapse-expand-icon {
        display: none !important;
      }
    `,
		icon: css$1`
      cursor: pointer;
      transition: all 100ms ${cssVar$1.motionEaseOut};
    `,
		outlined: css$1`
      &.${prefixCls}-collapse {
        border: 1px solid ${cssVar$1.colorFillSecondary};
        background: ${cssVar$1.colorBgContainer};
        .${prefixCls}-collapse-item .${prefixCls}-collapse-header {
          transition: none;
        }
        .${prefixCls}-collapse-item-active .${prefixCls}-collapse-header {
          border-block-end: 1px solid ${cssVar$1.colorFillTertiary};
        }
        .${prefixCls}-collapse-item:not(:first-child) {
          .${prefixCls}-collapse-header {
            border-block-start: 1px solid ${cssVar$1.colorFillTertiary};
          }
        }
      }
    `,
		root: css$1`
      &.${prefixCls}-collapse {
        display: flex;
        flex-direction: column;
        background: transparent;

        .${prefixCls}-collapse-header {
          overflow: hidden;
          display: flex;
          flex: none;
          gap: 0.75em;
          align-items: flex-start;

          border-radius: 0 !important;

          .${prefixCls}-collapse-header-text {
            flex: 1;
          }

          .${prefixCls}-collapse-expand-icon {
            align-items: center;
            min-height: 28px;
            margin: 0;
            padding: 0;
          }

          .${prefixCls}-collapse-extra {
            display: flex;
            align-items: center;
            min-height: 28px;
          }
        }

        .${prefixCls}-collapse-panel {
          background: transparent;
        }
      }
    `,
		title: css$1`
      font-size: 16px;
      font-weight: 500;
      line-height: 28px;
    `
	};
});
const variants = cva(styles.root, {
	compoundVariants: [
		{
			class: styles.gapOutlined,
			gap: true,
			variant: "outlined"
		},
		{
			class: styles.filledDark,
			isDarkMode: true,
			variant: "filled"
		},
		{
			class: styles.filledLight,
			isDarkMode: false,
			variant: "filled"
		}
	],
	defaultVariants: {
		collapsible: true,
		gap: false,
		isDarkMode: false
	},
	variants: {
		collapsible: {
			false: styles.hideCollapsibleIcon,
			true: null
		},
		gap: {
			false: null,
			true: styles.gapRoot
		},
		isDarkMode: {
			false: null,
			true: null
		},
		variant: {
			borderless: styles.borderless,
			filled: null,
			outlined: styles.outlined
		}
	}
});

//#endregion
export { DEFAULT_PADDING, getPadding, styles, variants };
//# sourceMappingURL=style.mjs.map