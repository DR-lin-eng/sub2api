import { staticStylish } from "../styles/theme/customStylishStatic.mjs";
import { createStaticStyles, cx } from "antd-style";

//#region src/SearchBar/style.ts
const styles = createStaticStyles(({ css: css$1, cssVar: cssVar$1 }) => ({
	icon: css$1`
    color: ${cssVar$1.colorTextPlaceholder};
  `,
	search: css$1`
    position: relative;
    max-width: 100%;
  `,
	tag: cx(staticStylish.blur, css$1`
      position: absolute;
      inset-block-start: 50%;
      inset-inline-end: 6px;
      transform: translateY(-50%);

      color: ${cssVar$1.colorTextDescription};

      kbd {
        color: inherit;
      }
    `)
}));

//#endregion
export { styles };
//# sourceMappingURL=style.mjs.map