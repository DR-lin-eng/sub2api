import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, cx, responsive } from "antd-style";

//#region src/Toc/style.ts
const prefixCls = "ant";
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		anchor: cx(staticStylish.blur, css$1`
        overflow: hidden auto;
        max-height: 60dvh !important;
      `),
		container: css$1`
      position: sticky;
      inset-block-start: calc(var(--toc-header-height, 64px) + 64px);

      overscroll-behavior: contain;
      grid-area: toc;

      width: var(--toc-width, 176px);
      margin-inline-end: 24px;
      border-radius: 3px;

      -webkit-overflow-scrolling: touch;

      ${responsive.sm} {
        position: relative;
        inset-inline-start: 0;
        width: 100%;
        margin-block-start: 0;
      }

      > h4 {
        margin-block: 0 8px;
        margin-inline: 0;

        font-size: 12px;
        line-height: 1;
        color: ${cssVar$1.colorTextDescription};
      }
    `,
		expand: cx(staticStylish.blur, css$1`
        width: 100%;
        border-block-end: 1px solid ${cssVar$1.colorSplit};
        border-radius: 0;

        background-color: color-mix(in srgb, ${cssVar$1.colorBgLayout} 50%, transparent);
        box-shadow: ${cssVar$1.boxShadowSecondary};

        .${prefixCls}-collapse-content {
          overflow: auto;
        }

        .${prefixCls}-collapse-header {
          z-index: 10;
          padding-block: 8px !important;
          padding-inline: 16px 8px !important;
        }
      `),
		mobileCtn: css$1`
      width: 100%;
      height: ${36}px;

      .${prefixCls}-collapse-expand-icon {
        color: ${cssVar$1.colorTextQuaternary};
      }
    `
	};
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map