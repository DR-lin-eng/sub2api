import { createStaticStyles } from "antd-style";
import { cva } from "class-variance-authority";

//#region src/Tabs/style.ts
const prefixCls = "ant";
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		compact: css$1`
      &.${prefixCls}-tabs {
        .${prefixCls}-tabs-tab {
          margin: 4px;

          + [class*='ant-tabs-tab'] {
            margin: 4px;
          }
        }
      }
    `,
		dropdown: css$1`
      .${prefixCls}-tabs-dropdown-menu {
        padding: 4px;
        border: 1px solid ${cssVar$1.colorBorderSecondary};

        .${prefixCls}-tabs-dropdown-menu-item {
          border-radius: ${cssVar$1.borderRadius};
        }
      }
    `,
		hideHolder: css$1`
      &.${prefixCls}-tabs {
        .${prefixCls}-tabs-content-holder {
          display: none;
        }

        .${prefixCls}-tabs-nav {
          margin: 0;

          &::before {
            display: none;
          }
        }
      }
    `,
		margin: css$1`
      &.${prefixCls}-tabs {
        .${prefixCls}-tabs-tab {
          margin: 8px;

          + .${prefixCls}-tabs-tab {
            margin: 8px;
          }
        }
      }
    `,
		point: css$1`
      &.${prefixCls}-tabs {
        &.${prefixCls}-tabs-top {
          .${prefixCls}-tabs-ink-bar {
            width: 8px !important;
            height: 4px;
            border-start-start-radius: 4px;
            border-start-end-radius: 4px;
          }
        }

        &.${prefixCls}-tabs-bottom {
          .${prefixCls}-tabs-ink-bar {
            width: 8px !important;
            height: 4px;
            border-end-start-radius: 4px;
            border-end-end-radius: 4px;
          }
        }

        &.${prefixCls}-tabs-left {
          .${prefixCls}-tabs-ink-bar {
            width: 4px;
            height: 8px !important;
            border-start-start-radius: 4px;
            border-end-start-radius: 4px;
          }
        }

        &.${prefixCls}-tabs-right {
          .${prefixCls}-tabs-ink-bar {
            width: 4px;
            height: 8px !important;
            border-start-end-radius: 4px;
            border-end-end-radius: 4px;
          }
        }
      }
    `,
		root: css$1`
      &.${prefixCls}-tabs {
        .${prefixCls}-tabs-tab {
          padding-block: 8px;
          padding-inline: 12px;
          color: ${cssVar$1.colorTextSecondary};
          transition: background-color 100ms ease-out;

          &:hover {
            border-radius: ${cssVar$1.borderRadius};
            color: ${cssVar$1.colorText};
            background: ${cssVar$1.colorFillTertiary};
          }
        }
      }
    `,
		rounded: css$1`
      &.${prefixCls}-tabs {
        &.${prefixCls}-tabs-top {
          .${prefixCls}-tabs-ink-bar {
            height: 3px;
            border-start-start-radius: 3px;
            border-start-end-radius: 3px;
          }
        }

        &.${prefixCls}-tabs-bottom {
          .${prefixCls}-tabs-ink-bar {
            height: 3px;
            border-end-start-radius: 3px;
            border-end-end-radius: 3px;
          }
        }

        &.${prefixCls}-tabs-left {
          .${prefixCls}-tabs-ink-bar {
            width: 3px;
            border-start-start-radius: 3px;
            border-end-start-radius: 3px;
          }
        }

        &.${prefixCls}-tabs-right {
          .${prefixCls}-tabs-ink-bar {
            width: 3px;
            border-start-end-radius: 3px;
            border-end-end-radius: 3px;
          }
        }
      }
    `
	};
});
const variants = cva(styles.root, {
	defaultVariants: {
		compact: false,
		underlined: false,
		variant: "rounded"
	},
	variants: {
		variant: {
			square: null,
			rounded: styles.rounded,
			point: styles.point
		},
		compact: {
			false: styles.margin,
			true: styles.compact
		},
		underlined: {
			false: styles.hideHolder,
			true: null
		}
	}
});

//#endregion
export { styles, variants };
//# sourceMappingURL=style.mjs.map