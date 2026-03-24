import { createStaticStyles } from "antd-style";

//#region src/FormModal/style.ts
const prefixCls = "ant";
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1, responsive: responsive$1 }) => ({
	footer: css$1`
    position: absolute;
    z-index: 10;
    inset-block-end: 0;
    inset-inline: 0;

    width: 100%;
    margin: 0;
    padding: 16px;

    background: linear-gradient(
      to bottom,
      color-mix(in srgb, ${cssVar$1.colorBgContainer} 0%, transparent) 0%,
      ${cssVar$1.colorBgContainer} 10%
    );
  `,
	form: css$1`
    position: static;
    .${prefixCls}-form-group-title {
      font-size: 15px;
      font-weight: 500;
    }

    ${responsive$1.sm} {
      .${prefixCls}-form-group-title {
        font-size: 14px;
        font-weight: normal;
      }

      .${prefixCls}-form-group {
        width: calc(100% + 32px);
        margin-inline: -16px;
        background: transparent;
      }
    }
  `
}));

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map