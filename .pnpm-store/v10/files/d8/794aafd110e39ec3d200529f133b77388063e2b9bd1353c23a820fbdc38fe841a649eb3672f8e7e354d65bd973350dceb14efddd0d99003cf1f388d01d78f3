import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, responsive } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Form/style.ts
const prefixCls = "ant";
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => ({
	borderless: css$1`
    gap: 48px;
    .${prefixCls}-collapse .${prefixCls}-collapse-header {
      padding-block-end: 16px;
      border-block-end: 1px solid ${cssVar$1.colorBorderSecondary};
    }

    .${prefixCls}-collapse-body {
      padding-inline: 0 !important;
    }
  `,
	filled: css$1`
    .${prefixCls}-collapse-body {
      padding-block: 0 !important;
    }
  `,
	outlined: css$1`
    .${prefixCls}-collapse-body {
      padding-block: 0 !important;
    }
  `,
	root: css$1`
    position: relative;

    display: flex;
    flex-direction: column;
    gap: 16px;

    width: 100%;

    .${prefixCls}-form-item {
      margin: 0 !important;
    }

    .${prefixCls}-form-item .${prefixCls}-form-item-label > label {
      height: unset;
    }

    .${prefixCls}-row {
      position: relative;
      flex-wrap: nowrap;
    }

    .${prefixCls}-form-item-label {
      position: relative;
      flex: 1;
      max-width: 100%;
    }

    .${prefixCls}-form-item-row {
      align-items: center;
    }

    .${prefixCls}-form-item-control {
      position: relative;
      flex: 0;
      min-width: unset !important;
    }

    .${prefixCls}-collapse-item {
      border-radius: ${cssVar$1.borderRadius} !important;
    }

    ${responsive.sm} {
      gap: 0 !important;
    }
  `
}));
const variants = cva(styles.root, {
	defaultVariants: { variant: "borderless" },
	variants: { variant: {
		filled: styles.filled,
		outlined: styles.outlined,
		borderless: styles.borderless
	} }
});
const flatGroupStyles = createStaticStyles(({ cx: cx$1, css: css$1, cssVar: cssVar$1 }) => {
	return {
		borderless: cx$1(staticStylish.variantBorderlessWithoutHover, css$1`
        padding-inline: 0;
      `),
		filled: cx$1(staticStylish.variantFilledWithoutHover, css$1`
        background: ${cssVar$1.colorFillQuaternary};
      `),
		mobile: css$1`
      padding-block: 0;
      padding-inline: 16px;
      border-radius: 0;
      background: ${cssVar$1.colorBgContainer};
    `,
		outlined: staticStylish.variantOutlinedWithoutHover,
		root: css$1`
      padding-inline: 16px;
      border-radius: ${cssVar$1.borderRadiusLG};
    `
	};
});
const flatGroupVariants = cva(flatGroupStyles.root, {
	defaultVariants: { variant: "borderless" },
	variants: { variant: {
		filled: flatGroupStyles.filled,
		outlined: flatGroupStyles.outlined,
		borderless: flatGroupStyles.borderless
	} }
});
const footerStyles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return { root: css$1`
      ${responsive.sm} {
        padding: 16px;
        border-block-start: 1px solid ${cssVar$1.colorBorderSecondary};
        background: ${cssVar$1.colorBgContainer};
      }
    ` };
});
const groupStyles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		mobileGroupBody: css$1`
      padding-block: 0;
      padding-inline: 16px;
      background: ${cssVar$1.colorBgContainer};
    `,
		mobileGroupHeader: css$1`
      padding: 16px;
      background: ${cssVar$1.colorBgLayout};
    `,
		title: css$1`
      align-items: center;
      font-size: 16px;
      font-weight: bold;
    `,
		titleBorderless: css$1`
      font-size: 18px;
      font-weight: bold;
    `,
		titleMobile: css$1`
      ${responsive.sm} {
        font-size: 14px;
        font-weight: 400;
        opacity: 0.5;
      }
    `
	};
});
const titleVariants = cva(groupStyles.title, {
	defaultVariants: { variant: "borderless" },
	variants: { variant: {
		filled: null,
		outlined: null,
		borderless: groupStyles.titleBorderless
	} }
});
const itemStyles = createStaticStyles(({ css: css$1 }) => ({
	itemMinWidth: css$1`
    &.${prefixCls}-form-item .${prefixCls}-form-item-control {
      width: var(--form-item-min-width) !important;
    }
  `,
	itemNoDivider: css$1`
    &:not(:first-child) {
      padding-block-start: 0;
    }
  `,
	root: css$1`
    &.${prefixCls}-form-item {
      padding-block: 16px;
      padding-inline: 0;

      .${prefixCls}-form-item-label {
        text-align: start;
      }

      .${prefixCls}-row {
        gap: 12px;
        justify-content: space-between;

        > div {
          flex: unset;
          flex-grow: unset;
        }
      }

      .${prefixCls}-form-item-required::before {
        align-self: flex-start;
      }

      ${responsive.sm} {
        &.${prefixCls}-form-item-horizontal {
          .${prefixCls}-form-item-label {
            flex: 1 !important;
          }
          .${prefixCls}-form-item-control {
            flex: none !important;
          }
        }
      }
    }
  `,
	verticalLayout: css$1`
    &.${prefixCls}-form-item {
      .${prefixCls}-row {
        align-items: stretch;
      }
    }
  `
}));
const itemVariants = cva(itemStyles.root, {
	defaultVariants: {
		divider: false,
		itemMinWidth: false,
		layout: "vertical"
	},
	variants: {
		itemMinWidth: {
			true: itemStyles.itemMinWidth,
			false: null
		},
		divider: {
			true: null,
			false: itemStyles.itemNoDivider
		},
		layout: {
			vertical: itemStyles.verticalLayout,
			horizontal: null
		}
	}
});
const submitFooterStyles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => ({
	floatFooter: css$1`
    position: fixed;
    z-index: 1000;
    inset-block-end: 24px;
    inset-inline-start: 50%;
    transform: translateX(-50%);

    width: max-content;
    padding: 8px;
    border: 1px solid ${cssVar$1.colorBorderSecondary};
    border-radius: 48px;

    background: ${cssVar$1.colorBgContainer};
    box-shadow: ${cssVar$1.boxShadowSecondary};
  `,
	footer: css$1`
    ${responsive.sm} {
      margin-block-start: calc(-1 * ${cssVar$1.borderRadius});
      padding: 16px;
      border-block-start: 1px solid ${cssVar$1.colorBorderSecondary};
      background: ${cssVar$1.colorBgContainer};
    }
  `
}));
const titleStyles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => ({
	content: css$1`
    position: relative;
    text-align: start;
  `,
	desc: css$1`
    display: block;

    line-height: 1.44;
    color: ${cssVar$1.colorTextDescription};
    word-wrap: break-word;
    white-space: pre-wrap;
  `,
	title: css$1`
    font-weight: 500;
    line-height: 1;
  `
}));

//#endregion
export { flatGroupStyles, flatGroupVariants, groupStyles, itemVariants, submitFooterStyles, titleStyles, titleVariants, variants };
//# sourceMappingURL=style.mjs.map