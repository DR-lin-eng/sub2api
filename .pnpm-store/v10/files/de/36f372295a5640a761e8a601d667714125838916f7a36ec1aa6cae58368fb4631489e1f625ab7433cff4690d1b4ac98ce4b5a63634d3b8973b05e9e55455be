import { staticStylish } from "../../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, cx } from "antd-style";

//#region src/chat/ChatInputArea/style.ts
const styles = createStaticStyles(({ css: css$1 }) => {
	return {
		container: css$1`
      position: relative;

      display: flex;
      flex-direction: column;
      gap: 8px;

      height: 100%;
      padding-block: 8px 12px;
      padding-inline: 0;
    `,
		textarea: css$1`
      height: 100% !important;
      padding-block: 0;
      padding-inline: 8px;
      line-height: 1.5;
    `,
		textareaContainer: css$1`
      position: relative;
      flex: 1;
    `
	};
});
const actionBarStyles = createStaticStyles(({ css: css$1 }) => ({
	left: cx(staticStylish.noScrollbar, css$1`
      overflow: auto hidden;
    `),
	right: css$1``,
	root: css$1`
    position: relative;
    overflow: hidden;
    width: 100%;
  `
}));

//#endregion
export { actionBarStyles, styles };
//# sourceMappingURL=style.mjs.map