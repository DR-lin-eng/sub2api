import { createStaticStyles } from "antd-style";

//#region src/Header/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1, responsive: responsive$1 }) => ({
	left: css$1`
    z-index: 10;
  `,
	right: css$1`
    z-index: 10;

    &-aside {
      display: flex;
      align-items: center;

      ${responsive$1.sm} {
        justify-content: center;

        margin-block: 8px;
        margin-inline: 16px;
        padding-block-start: 24px;
        border-block-start: 1px solid ${cssVar$1.colorBorder};
      }
    }
  `,
	root: css$1`
    grid-area: head;
    align-self: stretch;

    width: 100%;
    height: 64px;
    padding-block: 0;
    padding-inline: 24px;
    border-block-end: 1px solid ${cssVar$1.colorBorderSecondary};

    background-color: color-mix(in srgb, ${cssVar$1.colorBgLayout} 40%, transparent);

    ${responsive$1.sm} {
      padding-block: 0;
      padding-inline: 12px;
    }
  `
}));

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map