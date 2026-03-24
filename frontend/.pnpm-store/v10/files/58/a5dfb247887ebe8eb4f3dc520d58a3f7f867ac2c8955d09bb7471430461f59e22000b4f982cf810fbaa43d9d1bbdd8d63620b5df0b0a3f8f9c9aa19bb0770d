import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, cx } from "antd-style";

//#region src/Layout/style.ts
const styles = createStaticStyles(({ css: css$1 }) => {
	return {
		app: css$1`
      overflow: hidden auto;
      height: 100dvh;
    `,
		aside: css$1`
      position: sticky;
      z-index: 2;
      height: 100%;
    `,
		asideInner: css$1`
      overflow: hidden auto;
      width: 100%;
      height: calc(100dvh - var(--layout-header-height, 64px));
    `,
		content: css$1`
      position: relative;
      flex: 1;
      max-width: 100%;
    `,
		footer: css$1`
      position: relative;
      max-width: 100%;
    `,
		header: cx(staticStylish.blur, css$1`
        position: sticky;
        z-index: 999;
        inset-block-start: 0;
        max-width: 100%;
      `),
		main: css$1`
      position: relative;
      display: flex;
      align-items: stretch;
      max-width: 100vw;
    `,
		toc: css$1``
	};
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map