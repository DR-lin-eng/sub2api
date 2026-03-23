import { createStaticStyles } from "antd-style";

//#region src/Modal/style.ts
const HEADER_HEIGHT = 56;
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		content: css$1`
      [class*='ant-modal-footer'] {
        margin: 0;
        padding: 16px;
      }

      [class*='ant-modal-header'] {
        display: flex;
        gap: 4px;
        align-items: center;
        justify-content: center;

        height: 56px;
        margin-block-end: 0;
        padding: 16px;
      }

      [class*='ant-modal-container'] {
        overflow: hidden;
        padding: 0;
        border: 1px solid ${cssVar$1.colorSplit};
        border-radius: ${cssVar$1.borderRadiusLG};
      }
    `,
		drawerContent: css$1`
      [class*='ant-drawer-close'] {
        padding: 0;
      }

      [class*='ant-drawer-header'] {
        flex: none;
        height: ${HEADER_HEIGHT}px !important;
        padding-block: 0;
        padding-inline: 16px;
      }

      [class*='ant-drawer-footer'] {
        display: flex;
        align-items: center;
        justify-content: flex-end;

        padding: 16px;
        border: none;
      }
    `,
		wrap: css$1`
      overflow: hidden auto;
    `
	};
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map