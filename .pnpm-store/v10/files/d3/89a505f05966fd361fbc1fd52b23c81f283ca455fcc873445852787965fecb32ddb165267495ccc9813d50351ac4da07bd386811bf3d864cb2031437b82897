import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles } from "antd-style";

//#region src/Burger/style.ts
const prefixCls = "ant";
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		container: css$1`
      cursor: pointer;
      width: ${cssVar$1.controlHeight};
      height: ${cssVar$1.controlHeight};
      border-radius: ${cssVar$1.borderRadius};
    `,
		drawer: css$1`
      &.${prefixCls}-drawer-content {
        background: transparent;
      }

      .${prefixCls}-drawer-body {
        display: flex;
        flex-direction: column;
      }
    `,
		drawerRoot: css$1`
      inset-block-start: var(--burger-drawer-top, calc(var(--burger-header-height, 64px) + 1px));

      :focus-visible {
        outline: none;
      }

      .${prefixCls}-drawer {
        &-mask {
          ${staticStylish.blur};
          background-color: color-mix(in srgb, ${cssVar$1.colorBgLayout} 50%, transparent);
        }

        &-content-wrapper {
          box-shadow: none;
        }
      }
    `,
		drawerRootFullscreen: css$1`
      inset-block-start: 0;

      :focus-visible {
        outline: none;
      }

      .${prefixCls}-drawer {
        &-mask {
          ${staticStylish.blur};
          background-color: color-mix(in srgb, ${cssVar$1.colorBgLayout} 50%, transparent);
        }

        &-content-wrapper {
          box-shadow: none;
        }
      }
    `,
		fillRect: css$1`
      flex: 1;
      width: 100%;
      border-block-start: none;
    `,
		menu: css$1`
      padding-block-start: var(--burger-menu-padding-top, 0);
      border-inline-end: transparent !important;
      background: transparent;

      > .${prefixCls}-menu-item-only-child, .${prefixCls}-menu-submenu-title {
        width: 100%;
        margin: 0 !important;
        border-block-start: none;
        border-radius: 0;

        &:active {
          color: ${cssVar$1.colorText};
          background-color: ${cssVar$1.colorFill};
        }
      }

      .${prefixCls}-menu-item-only-child:first-child {
        border-block-start: none;
      }

      .${prefixCls}-menu-submenu-title[aria-expanded='true'] {
        a {
          font-weight: 600;
          color: ${cssVar$1.colorText} !important;
        }
      }

      .${prefixCls}-menu-item-group-title {
        padding-block: 8px;

        font-size: 12px;
        font-weight: 500;
        line-height: 1;
        text-overflow: ellipsis;
        text-transform: uppercase;
        white-space: nowrap;

        background: ${cssVar$1.colorFillSecondary};
      }

      .${prefixCls}-menu-item {
        width: calc(100% - 16px) !important;
        margin: 8px !important;
        border-radius: ${cssVar$1.borderRadius};

        &:hover,
        &:active {
          color: ${cssVar$1.colorText} !important;
          background: ${cssVar$1.colorFillSecondary} !important;
        }
      }

      .${prefixCls}-menu-item-active {
        width: calc(100% - 16px) !important;
        margin: 8px !important;
        border-radius: ${cssVar$1.borderRadius};
        background: ${cssVar$1.colorFillSecondary};
      }

      .${prefixCls}-menu-item-selected {
        font-weight: 600;
        color: ${cssVar$1.colorBgLayout};
        background: ${cssVar$1.colorText};

        &:hover,
        &:active {
          color: ${cssVar$1.colorBgLayout} !important;
          background: ${cssVar$1.colorText} !important;
        }
      }
    `
	};
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map