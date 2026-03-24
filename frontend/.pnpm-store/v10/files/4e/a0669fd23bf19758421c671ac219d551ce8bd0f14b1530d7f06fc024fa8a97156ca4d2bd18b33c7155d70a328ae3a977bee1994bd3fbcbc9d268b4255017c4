import { staticStylish } from "../../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, cx } from "antd-style";

//#region src/List/ListItem/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => {
	return {
		actions: css$1`
      position: absolute;
      inset-block-start: 50%;
      inset-inline-end: 16px;
      transform: translateY(-50%);
    `,
		active: staticStylish.active,
		content: css$1`
      position: relative;
      overflow: hidden;
      flex: 1;
      align-self: center;
    `,
		date: css$1`
      font-size: 12px;
      color: ${cssVar$1.colorTextPlaceholder};
    `,
		desc: css$1`
      width: 100%;
      margin: 0;

      font-size: 12px;
      line-height: 1.2;
      color: ${cssVar$1.colorTextDescription};
    `,
		pin: css$1`
      position: absolute;
      inset-block-start: 6px;
      inset-inline-end: 6px;
    `,
		root: cx(staticStylish.variantBorderless, css$1`
        cursor: pointer;
        position: relative;
        border-radius: ${cssVar$1.borderRadius};
        color: ${cssVar$1.colorTextTertiary};
      `),
		title: css$1`
      width: 100%;
      margin: 0;

      font-size: 14px;
      font-weight: 500;
      line-height: 1.2;
      color: ${cssVar$1.colorText};
    `,
		triangle: css$1`
      width: 10px;
      height: 10px;
      border-radius: 2px;

      opacity: 0.5;
      background: ${cssVar$1.colorPrimaryBorder};
      clip-path: polygon(0% 0%, 100% 0%, 100% 100%);
    `
	};
});

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map